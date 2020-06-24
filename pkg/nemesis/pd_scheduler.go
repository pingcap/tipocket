package nemesis

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/util/pdutil"
)

// schedulerGenerator generates shuffle-leader-scheduler,shuffle-region-scheduler
type schedulerGenerator struct {
	name string
}

// NewSchedulerGenerator ...
func NewSchedulerGenerator(name string) *schedulerGenerator {
	return &schedulerGenerator{
		name: name,
	}
}

func (s *schedulerGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	duration := time.Duration(5) * time.Minute
	return s.schedule(nodes, duration)
}

func (s *schedulerGenerator) Name() string {
	return s.name
}

func (s *schedulerGenerator) schedule(nodes []cluster.Node, duration time.Duration) []*core.NemesisOperation {
	nodes = filterComponent(nodes, cluster.PD)
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

type scheduler struct{}

func (s scheduler) Invoke(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute*time.Duration(10))
	defer cancel()

	schedulerName := extractSchedulerArgs(args...)
	pdAddr := fmt.Sprintf("http://%s:%d", node.IP, node.Port)
	client := pdutil.NewPDClient(http.DefaultClient, pdAddr)
	log.Infof("apply nemesis %s %s on ns %s", core.PDScheduler, schedulerName, node.Namespace)
	if err := client.AddScheduler(schedulerName); err != nil {
		log.Errorf("pd scheduling error: %+v", err)
	} else {
		log.Infof("add scheduler %s successfully", schedulerName)
	}
	return nil
}

func (s scheduler) Recover(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute*time.Duration(10))
	defer cancel()

	schedulerName := extractSchedulerArgs(args...)
	pdAddr := fmt.Sprintf("http://%s:%d", node.IP, node.Port)
	client := pdutil.NewPDClient(http.DefaultClient, pdAddr)
	log.Infof("unapply nemesis %s %s on ns %s", core.PDScheduler, schedulerName, node.Namespace)
	if err := client.RemoveScheduler(schedulerName); err != nil {
		log.Errorf("pd scheduling error: %+v", err)
	} else {
		log.Infof("remove scheduler %s successfully", schedulerName)
	}
	return nil
}

func (s scheduler) Name() string {
	return string(core.PDScheduler)
}

func extractSchedulerArgs(args ...interface{}) string {
	return args[0].(string)
}
