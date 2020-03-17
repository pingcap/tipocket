package nemesis

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/util/pdutil"
)

// schedulerGenerator generates shuffle-leader-scheduler,shuffle-region-scheduler
type schedulerGenerator struct {
	name      string
}

func NewSchedulerGenerator(name string) *schedulerGenerator {
	return &schedulerGenerator{
		name: name,
	}
}

func (s *schedulerGenerator) Generate(nodes []clusterTypes.Node) []*core.NemesisOperation {
	duration := time.Duration(5) * time.Minute
	return s.schedule(nodes, duration)
}

func (s *schedulerGenerator) Name() string {
	return s.name
}

func (s *schedulerGenerator) schedule(nodes []clusterTypes.Node, duration time.Duration) []*core.NemesisOperation {
	nodes = filterComponent(nodes, clusterTypes.PD)
	var ops []*core.NemesisOperation
	ops = append(ops, &core.NemesisOperation{
		Type:        core.PDScheduler,
		Node:        &nodes[0],
		InvokeArgs:  []interface{}{s.name},
		RecoverArgs: []interface{}{s.name},
		RunTime:     duration,
	})
	return ops
}

type scheduler struct {
}

func (s scheduler) Invoke(ctx context.Context, node *clusterTypes.Node, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute*time.Duration(10))
	defer cancel()

	schedulerName := extractSchedulerArgs(args...)
	pdAddr := fmt.Sprintf("http://%s:%d", node.IP, node.Port)
	client := pdutil.NewPDClient(http.DefaultClient, pdAddr)
	if err := client.AddScheduler(schedulerName); err != nil {
		log.Fatalf("pd scheduling error: %+v", err)
	}
	log.Printf("add scheduler %s successfully", schedulerName)
	return nil
}

func (s scheduler) Recover(ctx context.Context, node *clusterTypes.Node, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute*time.Duration(10))
	defer cancel()

	schedulerName := extractSchedulerArgs(args...)
	pdAddr := fmt.Sprintf("http://%s:%d", node.IP, node.Port)
	client := pdutil.NewPDClient(http.DefaultClient, pdAddr)
	if err := client.RemoveScheduler(schedulerName); err != nil {
		log.Fatalf("pd scheduling error: %+v", err)
	}
	log.Printf("remove scheduler %s successfully", schedulerName)
	return nil
}

func (s scheduler) Name() string {
	return string(core.PDScheduler)
}

func extractSchedulerArgs(args ...interface{}) string {
	return args[0].(string)
}
