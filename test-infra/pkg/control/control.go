package control

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/pingcap/tipocket/test-infra/pkg/core"
	"github.com/pingcap/tipocket/test-infra/pkg/history"
	"github.com/pingcap/tipocket/test-infra/pkg/verify"
)

// Controller controls the whole cluster. It sends request to the database,
// and also uses nemesis to disturb the cluster.
// Here have only 5 nodes, and the hosts are n1 - n5.
type Controller struct {
	cfg *Config

	clients []core.Client
	ctx     context.Context
	cancel  context.CancelFunc

	proc         int64
	requestCount int64

	suit verify.Suit
}

// NewController creates a controller.
func NewController(
	ctx context.Context,
	cfg *Config,
	clientCreator core.ClientCreator,
	verifySuit verify.Suit,
) *Controller {
	cfg.adjust()

	c := new(Controller)
	c.cfg = cfg
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.suit = verifySuit

	for _, node := range c.cfg.Nodes {
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
	// 	c.setUpDB()
	c.setUpClient()

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
		log.Printf("total request count %d", requestCount)

		n := len(c.cfg.Nodes)
		var clientWg sync.WaitGroup
		clientWg.Add(n)
		for i := 0; i < n; i++ {
			go func(i int) {
				defer clientWg.Done()
				c.onClientLoop(ctx, i, &requestCount, recorder)
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

	c.tearDownClient()
	// c.tearDownDB()
}

func (c *Controller) syncExec(f func(i int)) {
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

func (c *Controller) setUpClient() {
	log.Printf("begin to set up client")
	c.syncExec(func(i int) {
		client := c.clients[i]
		node := c.cfg.Nodes[i]
		log.Printf("begin to set up db client for node %s", node)
		if err := client.SetUp(c.ctx, c.cfg.Nodes, node); err != nil {
			log.Fatalf("set up db client for node %s failed %v", node, err)
		}
	})
}

func (c *Controller) tearDownClient() {
	log.Printf("begin to tear down client")
	c.syncExec(func(i int) {
		client := c.clients[i]
		node := c.cfg.Nodes[i]
		log.Printf("begin to tear down db client for node %s", node)
		if err := client.TearDown(c.ctx, c.cfg.Nodes, node); err != nil {
			log.Printf("tear down db client for node %s failed %v", node, err)
		}
	})
}

func (c *Controller) dumpState(ctx context.Context, recorder *history.Recorder) error {
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()

	for _, client := range c.clients {
		for _, node := range c.cfg.Nodes {
			log.Printf("begin to dump on node %s", node)
			sum, err := client.DumpState(ctx)
			if err == nil {
				recorder.RecordState(sum)
				return nil
			}
		}
	}
	return fmt.Errorf("fail to dump")
}

func (c *Controller) onClientLoop(
	ctx context.Context,
	i int,
	requestCount *int64,
	recorder *history.Recorder,
) {
	client := c.clients[i]
	node := c.cfg.Nodes[i]

	log.Printf("begin to run command on node %s", node)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	procID := atomic.AddInt64(&c.proc, 1)
	for atomic.AddInt64(requestCount, -1) >= 0 {
		request := client.NextRequest()

		if err := recorder.RecordRequest(procID, request); err != nil {
			log.Fatalf("record request %v failed %v", request, err)
		}

		log.Printf("%s: call %+v", node, request)
		response := client.Invoke(ctx, node, request)
		log.Printf("%s: return %+v", node, response)
		isUnknown := true
		if v, ok := response.(core.UnknownResponse); ok {
			isUnknown = v.IsUnknown()
		}

		if err := recorder.RecordResponse(procID, response); err != nil {
			log.Fatalf("record response %v failed %v", response, err)
		}

		// If Unknown, we need to use another process ID.
		if isUnknown {
			procID = atomic.AddInt64(&c.proc, 1)
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
