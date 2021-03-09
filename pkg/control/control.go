package control

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/history"
	"github.com/pingcap/tipocket/pkg/logs"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/util"

	// register nemesis
	_ "github.com/pingcap/tipocket/pkg/nemesis"
)

// Controller controls the whole cluster. It sends request to the database,
// and also uses nemesis to disturb the cluster.
// Here have only 5 nodes, and the hosts are n1 - n5.
type Controller struct {
	cfg *Config

	clients []core.Client

	// protect nemesisGenerators' reading and updating
	sync.RWMutex
	nemesisGenerators core.NemesisGenerators

	clientRequestGenerator func(ctx context.Context,
		client core.OnScheduleClientExtensions,
		node cluster.ClientNode,
		proc *int64,
		requestCount *int64,
		recorder *history.Recorder)

	ctx    context.Context
	cancel context.CancelFunc

	proc         int64
	requestCount int64

	suit       verify.Suit
	plugins    []Plugin
	logsClient logs.SearchLogClient
}

// NewController creates a controller.
func NewController(
	ctx context.Context,
	cfg *Config,
	clientCreator core.ClientCreator,
	nemesisGenerators core.NemesisGenerators,
	clientRequestGenerator func(ctx context.Context, client core.OnScheduleClientExtensions, node cluster.ClientNode, proc *int64, requestCount *int64, recorder *history.Recorder),
	verifySuit verify.Suit,
	plugins []Plugin,
	logsClient logs.SearchLogClient,
) *Controller {
	if db := core.GetDB(cfg.DB); db == nil {
		log.Fatalf("database %s is not registered", cfg.DB)
	}
	c := new(Controller)
	c.cfg = cfg
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.nemesisGenerators = nemesisGenerators
	c.clientRequestGenerator = clientRequestGenerator
	c.suit = verifySuit
	c.plugins = plugins
	c.logsClient = logsClient

	for _, node := range c.cfg.ClientNodes {
		c.clients = append(c.clients, clientCreator.Create(node))
	}
	log.Infof("start controller with %+v", cfg)
	return c
}

// Close closes the controller.
func (c *Controller) Close() {
	c.cancel()
}

// Run runs the controller.
func (c *Controller) Run() {
	switch c.cfg.Mode {
	case ModeStandard:
		c.TransferControlToClient()
	case ModeOnSchedule:
		c.RunClientOnSchedule()
	case ModeNemesisSequential:
		c.RunWithNemesisSequential()
	default:
		log.Fatalf("unhandled mode %s", c.cfg.Mode)
	}
}

func (c *Controller) setUpPlugin() {
	for _, plugin := range c.plugins {
		plugin.InitPlugin(c)
	}
}

// RunClientOnSchedule runs workload round by round, with nemesis injected seamlessly
// Nemesis and workload are running concurrently, nemesis won't pause when one round of workload is finished
func (c *Controller) RunClientOnSchedule() {
	c.setUpDB()
	c.setUpClient()
	c.setUpPlugin()

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
				c.clientRequestGenerator(ctx, c.clients[i].(core.OnScheduleClientExtensions), c.cfg.ClientNodes[i], &proc, &requestCount, recorder)
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
	c.setUpPlugin()
ENTRY:
	for {
		c.RLock()
		gen := c.nemesisGenerators
		c.RUnlock()
		if !gen.HasNext() {
			break
		}
		g := gen.Next()
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
					c.clientRequestGenerator(ctx, c.clients[i].(core.OnScheduleClientExtensions), c.cfg.ClientNodes[i], &proc, &requestCount, recorder)
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

// TransferControlToClient transfer control to client
func (c *Controller) TransferControlToClient() {
	c.setUpDB()
	c.setUpClient()
	c.setUpPlugin()

	var (
		nemesisWg     sync.WaitGroup
		g             errgroup.Group
		nCtx, nCancel = context.WithTimeout(context.WithValue(c.ctx, "control", c), c.cfg.RunTime*time.Duration(int64(c.cfg.RunRound)))
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
		ii := i
		g.Go(func() error {
			log.Infof("run client %d...", ii)
			return client.(core.StandardClientExtensions).Start(nCtx, c.cfg.ClientConfig, c.cfg.ClientNodes)
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

// UpdateNemesisGenerators updates nemesis generators
func (c *Controller) UpdateNemesisGenerators(gs core.NemesisGenerators) {
	c.Lock()
	defer c.Unlock()
	c.nemesisGenerators = gs
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
		ctx := context.WithValue(c.ctx, "control", c)
		if err := client.SetUp(ctx, c.cfg.Nodes, c.cfg.ClientNodes, i); err != nil {
			log.Fatalf("set up db client for node %s failed: %v", c.cfg.ClientNodes[i], err)
		}
	})
}

func (c *Controller) tearDownClient() {
	log.Infof("begin to tear down client")
	c.syncClientExec(func(i int) {
		client := c.clients[i]
		log.Infof("begin to tear down db client for node %s", c.cfg.ClientNodes[i])
		if err := client.TearDown(c.ctx, c.cfg.ClientNodes, i); err != nil {
			log.Infof("tear down db client for node %s failed: %v", c.cfg.ClientNodes[i], err)
		}
	})
}

func (c *Controller) dumpState(ctx context.Context, recorder *history.Recorder) error {
	var sum interface{}
	err := wait.PollImmediate(10*time.Second, time.Minute*time.Duration(30), func() (bool, error) {
		var err error
		for _, client := range c.clients {
			for _, node := range c.cfg.ClientNodes {
				sum, err = client.(core.OnScheduleClientExtensions).DumpState(ctx)
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
		log.Infof("fail to dump state, may try again later: %v", err)
		return false, nil
	})
	return err
}

func (c *Controller) dispatchNemesis(ctx context.Context) {
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
		}
		c.RLock()
		gens := c.nemesisGenerators
		c.RUnlock()

		if !gens.HasNext() {
			if !gens.Reset() {
				time.Sleep(time.Second)
			}
			continue
		}
		var (
			gen = gens.Next()
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
		time.Sleep(30 * time.Second)
		return
	}
	log.Infof("run nemesis %s...", op.String())
	if err := nemesis.Invoke(ctx, op.Node, op.InvokeArgs...); err != nil {
		// because we cannot ensure the nemesis wasn't injected, so we also will try to recover it later.
		log.Errorf("run nemesis %s failed: %v", op.String(), err)
	}

	if op.NemesisControl != nil {
		op.NemesisControl.WaitForRollback(ctx)
	} else {
		select {
		case <-time.After(op.RunTime):
		case <-ctx.Done():
		}
	}
	log.Infof("recover nemesis %s...", op.String())
	err := util.RunWithRetry(ctx, 3, 10*time.Second, func() error {
		return nemesis.Recover(context.TODO(), op.Node, op.RecoverArgs...)
	})
	if err != nil {
		log.Errorf("recover nemesis %s failed: %v", op.String(), err)
	}
}
