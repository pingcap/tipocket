package control

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/history"
	"github.com/pingcap/tipocket/pkg/verify"

	// register nemesis
	_ "github.com/pingcap/tipocket/pkg/nemesis"

	// register tidb
	_ "github.com/pingcap/tipocket/db/tidb"
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
) *Controller {
	if len(cfg.DB) == 0 {
		log.Fatalf("empty database")
	}

	if db := core.GetDB(cfg.DB); db == nil {
		log.Fatalf("database %s is not registered", cfg.DB)
	}

	c := new(Controller)
	c.cfg = cfg
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.nemesisGenerators = nemesisGenerators
	c.clientRequestGenerator = clientRequestGenerator
	c.suit = verifySuit

	for _, node := range c.cfg.ClientNodes {
		c.clients = append(c.clients, clientCreator.Create(node))
	}

	log.Printf("start controller with %+v", cfg)

	return c
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

	nctx, ncancel := context.WithTimeout(c.ctx, c.cfg.RunTime*time.Duration(int64(c.cfg.RunRound)))
	var nemesisWg sync.WaitGroup
	nemesisWg.Add(1)
	go func() {
		defer nemesisWg.Done()
		c.dispatchNemesis(nctx)
	}()

ROUND:
	for round := 1; round <= c.cfg.RunRound; round++ {
		log.Printf("round %d start ...", round)

		ctx, cancel := context.WithTimeout(c.ctx, c.cfg.RunTime)

		historyFile := fmt.Sprintf("%s.%d", c.cfg.History, round)
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
		log.Printf("total request count %d", requestCount)

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
			log.Printf("finish test")
			break ROUND
		default:
		}

		log.Printf("round %d finish", round)
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
ENTRY:
	for _, g := range c.nemesisGenerators {
		for round := 1; round <= c.cfg.RunRound; round++ {
			log.Printf("nemesis[%s] round %d start...", g.Name(), round)

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
			log.Printf("total request count %d", requestCount)

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
			log.Printf("nemesis[%s] round %d client requests done", g.Name(), round)
			cancel()
			nemesisWg.Wait()
			log.Printf("nemesis[%s] round %d nemesis done", g.Name(), round)
			recorder.Close()

			log.Printf("nemesis[%s] round %d begin to verify history file", g.Name(), round)
			c.suit.Verify(historyFile)
			select {
			case <-c.ctx.Done():
				log.Printf("finish test")
				break ENTRY
			default:
			}

			log.Printf("nemesis[%s] round %d finish", g.Name(), round)
		}
	}

	c.tearDownClient()
	c.tearDownDB()
}

// RunSelfScheduled runs the controller with self scheduled
func (c *Controller) RunSelfScheduled() {
	c.setUpDB()
	c.setUpClient()
	nctx, ncancel := context.WithTimeout(c.ctx, c.cfg.RunTime*time.Duration(int64(c.cfg.RunRound)))
	var nemesisWg sync.WaitGroup
	nemesisWg.Add(1)
	go func() {
		defer nemesisWg.Done()
		c.dispatchNemesis(nctx)
	}()

	// ctx, _ := context.WithTimeout(c.ctx, c.cfg.RunTime)
	// No matter how many clients are created, we only use one here
	// the real multi clients logic should handle by case itself
	err := c.clients[0].Start(nctx, c.cfg.CaseConfig, c.cfg.ClientNodes)
	if err != nil {
		log.Printf("case error %+v", err)
	}

	ncancel()

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
	log.Printf("begin to set up database")
	c.syncNodeExec(func(i int) {
		log.Printf("begin to set up database on %s", c.cfg.Nodes[i])
		db := core.GetDB(c.cfg.DB)
		err := db.SetUp(c.ctx, c.cfg.Nodes, c.cfg.Nodes[i])
		if err != nil {
			log.Fatalf("setup db %s at node %s failed %v", c.cfg.DB, c.cfg.Nodes[i], err)
		}
	})
}

func (c *Controller) tearDownDB() {
	log.Printf("begin to tear down database")
	c.syncNodeExec(func(i int) {
		log.Printf("being to tear down database on %s", c.cfg.Nodes[i])
		db := core.GetDB(c.cfg.DB)
		if err := db.TearDown(c.ctx, c.cfg.Nodes, c.cfg.Nodes[i]); err != nil {
			log.Printf("tear down db %s at node %s failed %v", c.cfg.DB, c.cfg.Nodes[i], err)
		}
	})
}

func (c *Controller) setUpClient() {
	log.Printf("begin to set up client")
	c.syncClientExec(func(i int) {
		client := c.clients[i]
		log.Printf("begin to set up db client for node %s", c.cfg.ClientNodes[i])
		if err := client.SetUp(c.ctx, c.cfg.ClientNodes, i); err != nil {
			log.Fatalf("set up db client for node %s failed %v", c.cfg.ClientNodes[i], err)
		}
	})
}

func (c *Controller) tearDownClient() {
	log.Printf("begin to tear down client")
	c.syncClientExec(func(i int) {
		client := c.clients[i]
		log.Printf("begin to tear down db client for node %s", c.cfg.ClientNodes[i])
		if err := client.TearDown(c.ctx, c.cfg.ClientNodes, i); err != nil {
			log.Printf("tear down db client for node %s failed %v", c.cfg.ClientNodes[i], err)
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
					log.Printf("begin to dump on node %s", node)
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

	if err != nil {
		return fmt.Errorf("fail to dump: %v", err)
	}
	return nil
}

func (c *Controller) dispatchNemesis(ctx context.Context) {
	if len(c.nemesisGenerators) == 0 {
		return
	}

	log.Printf("begin to run nemesis")
	var wg sync.WaitGroup
LOOP:
	for {
		for _, g := range c.nemesisGenerators {
			select {
			case <-ctx.Done():
				break LOOP
			default:
			}

			log.Printf("begin to run %s nemesis generator", g.Name())
			ops := g.Generate(c.cfg.Nodes)

			wg.Add(len(ops))
			for i := 0; i < len(ops); i++ {
				go c.onNemesisLoop(ctx, ops[i], &wg)
			}
			wg.Wait()
		}
	}
	log.Printf("stop to run nemesis")
}

func (c *Controller) dispatchNemesisWithRecord(ctx context.Context, g core.NemesisGenerator, recorder *history.Recorder) {
	var wg sync.WaitGroup
	ops := g.Generate(c.cfg.Nodes)

	wg.Add(len(ops))
	err := recorder.RecordInvokeNemesis(core.NemesisGeneratorRecord{Name: g.Name(), Ops: ops})
	if err != nil {
		log.Printf("record invoking nemesis %s failed: %v", g.Name(), err)
	}
	for i := 0; i < len(ops); i++ {
		go c.onNemesisLoop(ctx, ops[i], &wg)
	}
	wg.Wait()
	if err := recorder.RecordRecoverNemesis(g.Name()); err != nil {
		log.Printf("record recovering nemesis %s failed: %v", g.Name(), err)
	}
}

func (c *Controller) onNemesisLoop(ctx context.Context, op *core.NemesisOperation, wg *sync.WaitGroup) {
	defer wg.Done()

	if op == nil {
		return
	}

	nemesis := core.GetNemesis(string(op.Type))
	if nemesis == nil {
		log.Printf("nemesis %s is not registered", op.Type)
		return
	}

	if err := nemesis.Invoke(ctx, op.Node, op.InvokeArgs...); err != nil {
		log.Printf("run nemesis %s failed: %v", op.Type, err)
	}

	select {
	case <-time.After(op.RunTime):
	case <-ctx.Done():
	}
	if err := nemesis.Recover(context.TODO(), op.Node, op.RecoverArgs...); err != nil {
		log.Printf("recover nemesis %s failed: %v", op.Type, err)
	}
}
