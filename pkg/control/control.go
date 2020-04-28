package control

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/wait"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/history"
	"github.com/pingcap/tipocket/pkg/loki"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/util"

	// register nemesis
	_ "github.com/pingcap/tipocket/pkg/nemesis"

	// register tidb
	_ "github.com/pingcap/tipocket/db/tidb"
)

var (
	minTime = time.Unix(0, 0)
)

// Controller controls the whole cluster. It sends request to the database,
// and also uses nemesis to disturb the cluster.
// Here have only 5 nodes, and the hosts are n1 - n5.
type Controller struct {
	cfg *Config

	clients []core.Client

	nemesisGenerators      []core.NemesisGenerator
	clientRequestGenerator func(ctx context.Context,
		client core.Client,
		node clusterTypes.ClientNode,
		proc *int64,
		requestCount *int64,
		recorder *history.Recorder)

	ctx    context.Context
	cancel context.CancelFunc

	proc         int64
	requestCount int64

	// TODO(yeya24): make log service an interface
	lokiClient    *loki.Client
	logPath       string
	podLogFiles   map[string]*os.File
	lastQueryTime time.Time

	suit verify.Suit
}

// NewController creates a controller.
func NewController(
	ctx context.Context,
	cfg *Config,
	clientCreator core.ClientCreator,
	nemesisGenerators []core.NemesisGenerator,
	clientRequestGenerator func(ctx context.Context, client core.Client, node clusterTypes.ClientNode, proc *int64, requestCount *int64, recorder *history.Recorder),
	verifySuit verify.Suit,
	lokiCli *loki.Client,
	logPath string,
) (*Controller, error) {
	if db := core.GetDB(cfg.DB); db == nil {
		log.Fatalf("database %s is not registered", cfg.DB)
	}
	c := new(Controller)
	c.cfg = cfg
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.nemesisGenerators = nemesisGenerators
	c.clientRequestGenerator = clientRequestGenerator
	c.suit = verifySuit
	c.lokiClient = lokiCli
	c.logPath = logPath
	c.podLogFiles = make(map[string]*os.File)
	// init time stamp
	c.lastQueryTime = minTime

	if _, err := os.Stat(c.logPath); err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(c.logPath, os.ModePerm); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	for _, node := range c.cfg.ClientNodes {
		c.clients = append(c.clients, clientCreator.Create(node))
	}
	log.Infof("start controller with %+v", cfg)
	return c, nil
}

// Close closes the controller.
func (c *Controller) Close() {
	c.cancel()
}

// Run runs the controller.
func (c *Controller) Run() {
	switch c.cfg.Mode {
	case ModeMixed:
		c.RunMixed()
	case ModeSequential:
		c.RunWithNemesisSequential()
	case ModeSelfScheduled:
		c.RunSelfScheduled()
	default:
		log.Fatalf("unhandled mode %s", c.cfg.Mode)
	}
}

// RunMixed runs workload round by round, with nemesis injected seamlessly
// Nemesis and workload are running concurrently, nemesis won't pause when one round of workload is finished
func (c *Controller) RunMixed() {
	c.setUpDB()
	c.setUpClient()
	c.collectLogs()

	nctx, ncancel := context.WithTimeout(c.ctx, c.cfg.RunTime*time.Duration(int64(c.cfg.RunRound)))
	var nemesisWg sync.WaitGroup
	nemesisWg.Add(1)
	go func() {
		defer nemesisWg.Done()
		c.dispatchNemesis(nctx)
	}()

ROUND:
	for round := 1; round <= c.cfg.RunRound; round++ {
		log.Infof("round %d start ...", round)

		ctx, cancel := context.WithTimeout(c.ctx, c.cfg.RunTime)

		historyFile := fmt.Sprintf("%s.%d", c.cfg.History, round)
		recorder, err := history.NewRecorder(historyFile)
		if err != nil {
			log.Fatalf("prepare history failed %v", err)
		}

		if err := c.dumpState(ctx, recorder); err != nil {
			log.Fatalf("dump state failed, %v", err)
		}

		// requestCount for the round, shared by all clients.
		requestCount := int64(c.cfg.RequestCount)
		proc := c.proc
		log.Infof("total request count %d", requestCount)

		n := len(c.cfg.ClientNodes)
		var clientWg sync.WaitGroup
		clientWg.Add(n)
		for i := 0; i < n; i++ {
			go func(i int) {
				defer clientWg.Done()
				c.clientRequestGenerator(ctx, c.clients[i], c.cfg.ClientNodes[i], &proc, &requestCount, recorder)
			}(i)
		}

		clientWg.Wait()
		cancel()

		recorder.Close()
		c.suit.Verify(historyFile)

		select {
		case <-c.ctx.Done():
			log.Infof("finish test")
			break ROUND
		default:
		}

		log.Infof("round %d finish", round)
	}

	ncancel()
	nemesisWg.Wait()

	c.tearDownClient()
	c.tearDownDB()
}

// RunWithNemesisSequential runs nemesis sequential, with n round of workload running with each kind of nemesis.
// eg. nemesis1, round 1, round 2, ... round n ->
//		nemesis2, round 1, round 2, ... round n ->
//		... nemesis n, round 1, round 2, ... round n
func (c *Controller) RunWithNemesisSequential() {
	c.setUpDB()
	c.setUpClient()
	c.collectLogs()
ENTRY:
	for _, g := range c.nemesisGenerators {
		for round := 1; round <= c.cfg.RunRound; round++ {
			log.Infof("nemesis[%s] round %d start...", g.Name(), round)

			ctx, cancel := context.WithTimeout(c.ctx, c.cfg.RunTime)
			historyFile := fmt.Sprintf("%s.%s.%d", c.cfg.History, g.Name(), round)
			recorder, err := history.NewRecorder(historyFile)
			if err != nil {
				log.Fatalf("prepare history failed %v", err)
			}

			if err := c.dumpState(ctx, recorder); err != nil {
				log.Fatalf("dump state failed %v", err)
			}

			// requestCount for the round, shared by all clients.
			requestCount := int64(c.cfg.RequestCount)
			proc := c.proc
			log.Infof("total request count %d", requestCount)

			n := len(c.cfg.ClientNodes)
			var clientWg sync.WaitGroup
			clientWg.Add(n)
			for i := 0; i < n; i++ {
				go func(i int) {
					defer clientWg.Done()
					c.clientRequestGenerator(ctx, c.clients[i], c.cfg.ClientNodes[i], &proc, &requestCount, recorder)
				}(i)
			}

			var nemesisWg sync.WaitGroup
			nemesisWg.Add(1)
			go func() {
				defer nemesisWg.Done()
				c.dispatchNemesisWithRecord(ctx, g, recorder)
			}()

			clientWg.Wait()
			log.Infof("nemesis[%s] round %d client requests done", g.Name(), round)
			cancel()
			nemesisWg.Wait()
			log.Infof("nemesis[%s] round %d nemesis done", g.Name(), round)
			recorder.Close()

			log.Infof("nemesis[%s] round %d begin to verify history file", g.Name(), round)
			c.suit.Verify(historyFile)
			select {
			case <-c.ctx.Done():
				log.Infof("finish test")
				break ENTRY
			default:
			}
			log.Infof("nemesis[%s] round %d finish", g.Name(), round)
		}
	}

	c.tearDownClient()
	c.tearDownDB()
}

// RunSelfScheduled runs the controller with self scheduled
func (c *Controller) RunSelfScheduled() {
	c.setUpDB()
	c.setUpClient()
	c.collectLogs()

	var (
		nemesisWg     sync.WaitGroup
		g             errgroup.Group
		nCtx, nCancel = context.WithTimeout(c.ctx, c.cfg.RunTime*time.Duration(int64(c.cfg.RunRound)))
	)
	nemesisWg.Add(1)
	go func() {
		defer nemesisWg.Done()
		c.dispatchNemesis(nCtx)
	}()

	for i := 0; i < len(c.clients); i++ {
		// truncate more clients
		if i >= c.cfg.ClientCount {
			break
		}
		client := c.clients[i]
		g.Go(func() error {
			log.Infof("run client %d...", i)
			return client.Start(nCtx, c.cfg.ClientConfig, c.cfg.ClientNodes)
		})
	}
	if err := g.Wait(); err != nil {
		log.Fatalf("run client error, %+v", errors.ErrorStack(err))
	}

	// cancel nCtx and wait nemesis ended
	nCancel()
	nemesisWg.Wait()

	c.tearDownClient()
	c.tearDownDB()
}

func (c *Controller) syncClientExec(f func(i int)) {
	var wg sync.WaitGroup
	n := len(c.cfg.ClientNodes)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			f(i)
		}(i)
	}
	wg.Wait()
}

func (c *Controller) syncNodeExec(f func(i int)) {
	var wg sync.WaitGroup
	n := len(c.cfg.Nodes)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			f(i)
		}(i)
	}
	wg.Wait()
}

func (c *Controller) setUpDB() {
	log.Infof("begin to set up database")
	c.syncNodeExec(func(i int) {
		log.Infof("begin to set up database on %s", c.cfg.Nodes[i])
		db := core.GetDB(c.cfg.DB)
		err := db.SetUp(c.ctx, c.cfg.Nodes, c.cfg.Nodes[i])
		if err != nil {
			log.Fatalf("setup db %s at node %s failed %v", c.cfg.DB, c.cfg.Nodes[i], err)
		}
	})
}

func (c *Controller) tearDownDB() {
	log.Infof("begin to tear down database")
	c.syncNodeExec(func(i int) {
		log.Infof("being to tear down database on %s", c.cfg.Nodes[i])
		db := core.GetDB(c.cfg.DB)
		if err := db.TearDown(c.ctx, c.cfg.Nodes, c.cfg.Nodes[i]); err != nil {
			log.Infof("tear down db %s at node %s failed %v", c.cfg.DB, c.cfg.Nodes[i], err)
		}
	})
}

func (c *Controller) setUpClient() {
	log.Infof("begin to set up client")
	c.syncClientExec(func(i int) {
		client := c.clients[i]
		log.Infof("begin to set up db client for node %s", c.cfg.ClientNodes[i])
		if err := client.SetUp(c.ctx, c.cfg.ClientNodes, i); err != nil {
			log.Fatalf("set up db client for node %s failed %v", c.cfg.ClientNodes[i], err)
		}
	})
}

func (c *Controller) tearDownClient() {
	log.Infof("begin to tear down client")
	c.syncClientExec(func(i int) {
		client := c.clients[i]
		log.Infof("begin to tear down db client for node %s", c.cfg.ClientNodes[i])
		if err := client.TearDown(c.ctx, c.cfg.ClientNodes, i); err != nil {
			log.Infof("tear down db client for node %s failed %v", c.cfg.ClientNodes[i], err)
		}
	})
}

func (c *Controller) dumpState(ctx context.Context, recorder *history.Recorder) error {
	var err error
	var sum interface{}
	err = wait.PollImmediate(10*time.Second, time.Minute*time.Duration(10), func() (bool, error) {
		for _, client := range c.clients {
			for _, node := range c.cfg.ClientNodes {
				sum, err = client.DumpState(ctx)
				if err == nil {
					log.Infof("begin to dump on node %s", node)
					recorder.RecordState(sum)
					return true, nil
				}
				if ctx.Err() != nil {
					return false, ctx.Err()
				}
			}
		}
		return false, nil
	})

	return err
}

func (c *Controller) dispatchNemesis(ctx context.Context) {
	if len(c.nemesisGenerators) == 0 {
		return
	}
LOOP:
	for {
		for _, gen := range c.nemesisGenerators {
			select {
			case <-ctx.Done():
				break LOOP
			default:
			}
			var (
				ops = gen.Generate(c.cfg.Nodes)
				g   errgroup.Group
			)
			for i := 0; i < len(ops); i++ {
				op := ops[i]
				g.Go(func() error {
					c.onNemesis(ctx, op)
					return nil
				})
			}
			_ = g.Wait()
		}
	}
}

func (c *Controller) dispatchNemesisWithRecord(ctx context.Context, gen core.NemesisGenerator, recorder *history.Recorder) {
	var (
		ops = gen.Generate(c.cfg.Nodes)
		g   errgroup.Group
	)
	err := recorder.RecordInvokeNemesis(core.NemesisGeneratorRecord{Name: gen.Name(), Ops: ops})
	if err != nil {
		log.Infof("record invoking nemesis %s failed: %v", gen.Name(), err)
	}
	for i := 0; i < len(ops); i++ {
		op := ops[i]
		g.Go(func() error {
			c.onNemesis(ctx, op)
			return nil
		})
	}
	_ = g.Wait()
	if err := recorder.RecordRecoverNemesis(gen.Name()); err != nil {
		log.Infof("record recovering nemesis %s failed: %v", gen.Name(), err)
	}
}

func (c *Controller) onNemesis(ctx context.Context, op *core.NemesisOperation) {
	if op == nil {
		return
	}
	nemesis := core.GetNemesis(string(op.Type))
	if nemesis == nil {
		log.Errorf("nemesis %s is not registered", op.Type)
		return
	}
	log.Infof("run nemesis %s...", op.String())
	if err := nemesis.Invoke(ctx, op.Node, op.InvokeArgs...); err != nil {
		// because we cannot ensure the nemesis wasn't injected, so we also will try to recover it later.
		log.Errorf("run nemesis %s failed: %v", op.String(), err)
	}
	select {
	case <-time.After(op.RunTime):
	case <-ctx.Done():
	}
	log.Infof("recover nemesis %s...", op.String())
	err := util.RunWithRetry(ctx, 3, 10*time.Second, func() error {
		return nemesis.Recover(context.TODO(), op.Node, op.RecoverArgs...)
	})
	if err != nil {
		log.Errorf("recover nemesis %s failed: %v", op.String(), err)
	}
}

func (c *Controller) collectLogs() {
	if c.lokiClient == nil {
		return
	}

	longDur := time.Minute * 10
	shortDur := time.Minute * 1
	panicTicker := time.NewTicker(longDur)
	// log ticker is a ticker for collecting pod logs, runs every 4 minute
	logTicker := time.NewTicker(shortDur)

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				c.fetchTiDBClusterLogs(c.cfg.Nodes)
				logTicker.Stop()
				panicTicker.Stop()
				return
			case <-panicTicker.C:
				c.checkTiDBClusterPanic(longDur, c.cfg.Nodes)
			case <-logTicker.C:
				c.fetchTiDBClusterLogs(c.cfg.Nodes)
			}
		}
	}()
}

func (c *Controller) checkTiDBClusterPanic(dur time.Duration, nodes []clusterTypes.Node) {
	wg := &sync.WaitGroup{}
	to := time.Now()
	from := to.Add(-dur)

	for _, n := range nodes {
		switch n.Component {
		case clusterTypes.TiDB, clusterTypes.TiKV, clusterTypes.PD:
			wg.Add(1)
			var nonMatch []string
			if n.Component == clusterTypes.TiKV {
				nonMatch = []string{
					"panic-when-unexpected-key-or-data",
					"No space left on device",
				}
			}

			// containerName is added to be sure that we only fetch the logs of
			// the container we want, not included sidecar containers.
			go func(ns, podName, containerName string, nonMatch []string) {
				defer wg.Done()
				texts, err := c.lokiClient.FetchPodLogs(ns, podName, containerName,
					"panic", nonMatch, from, to, 1000, false)
				if err != nil {
					log.Infof("failed to fetch logs from loki for pod %s in ns %s", podName, ns)
				} else if len(texts) > 0 {
					log.Fatalf("%d panics occurred in ns: %s pod %s. Content: %v", len(texts), ns, podName, texts)
				}
			}(n.Namespace, n.PodName, string(n.Component), nonMatch)
		default:
		}
	}
	wg.Wait()
}

func (c *Controller) fetchTiDBClusterLogs(nodes []clusterTypes.Node) {
	wg := &sync.WaitGroup{}
	to := time.Now()

	for _, n := range nodes {
		switch n.Component {
		case clusterTypes.TiDB, clusterTypes.TiKV, clusterTypes.PD:
			wg.Add(1)

			// containerName is added to be sure that we only fetch the logs of
			// the container we want, not included sidecar containers.
			go func(ns, podName, containerName string) {
				defer wg.Done()
				texts, err := c.lokiClient.FetchPodLogs(ns, podName, containerName,
					"", nil, c.lastQueryTime, to, 20000, false)
				if err != nil {
					log.Infof("failed to fetch logs from loki for pod %s in ns %s", podName, ns)
				}
				log.Infof("collect %s pod logs successfully, %d lines total", podName, len(texts))
				if _, ok := c.podLogFiles[podName]; !ok {
					c.podLogFiles[podName], err = os.OpenFile(path.Join(c.logPath, podName),
						os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
					if err != nil {
						log.Fatalf("failed to create log file for pod %s error is %v", podName, err)
					}
				}
				for _, line := range texts {
					if _, err = c.podLogFiles[podName].Write([]byte(line)); err != nil {
						log.Fatal("fail to write log file for pod %s error is %v", podName, err)
					}
				}
			}(n.Namespace, n.PodName, string(n.Component))
		default:
		}
	}
	wg.Wait()
	c.lastQueryTime = to
}
