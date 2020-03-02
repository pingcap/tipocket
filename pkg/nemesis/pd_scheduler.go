package nemesis

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
)

// schedulerGenerator generates shuffle-leader-scheduler,shuffle-region-scheduler
type schedulerGenerator struct {
	name      string
	installed bool
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

func (s *schedulerGenerator) installPDCtl(version string) {
	if s.installed {
		return
	}
	// TODO validate version format
	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute*time.Duration(10))
	defer cancel()

	log.Printf("installing pd-ctl version %s", version)
	cmdStr := "curl http://download.pingcap.org/tidb-" + version + "-linux-amd64.tar.gz | tar xz --strip-components=2 -C /bin"
	if err := exec.CommandContext(ctx, "sh", "-c", cmdStr).Run(); err != nil {
		log.Fatalf("installing pd-ctl error: %+v", cmdStr)
	}
	s.installed = true
}

func (s *schedulerGenerator) schedule(nodes []clusterTypes.Node, duration time.Duration) []*core.NemesisOperation {
	nodes = filterComponent(nodes, clusterTypes.PD)
	s.installPDCtl(nodes[0].Version)
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

type Scheduler struct {
}

func (s Scheduler) Invoke(ctx context.Context, node *clusterTypes.Node, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute*time.Duration(10))
	defer cancel()

	schedulerName := extractSchedulerArgs(args...)
	cmdStr := fmt.Sprintf("pd-ctl -u http://%s:%d scheduler add %s", node.IP, node.Port, schedulerName)
	out, err := exec.CommandContext(ctx, "sh", "-c", cmdStr).CombinedOutput()
	if err != nil {
		log.Fatalf("pd scheduling error: %+v", err)
	}
	log.Printf("%s result: %s", cmdStr, string(out))
	return nil
}

func (s Scheduler) Recover(ctx context.Context, node *clusterTypes.Node, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute*time.Duration(10))
	defer cancel()

	schedulerName := extractSchedulerArgs(args...)
	cmdStr := fmt.Sprintf("pd-ctl -u http://%s:%d scheduler remove %s", node.IP, node.Port, schedulerName)
	out, err := exec.CommandContext(ctx, "sh", "-c", cmdStr).CombinedOutput()
	if err != nil {
		log.Fatalf("pd scheduling error: %+v", err)
	}
	log.Printf("%s result: %s", cmdStr, string(out))
	return nil
}

func (s Scheduler) Name() string {
	return string(core.PDScheduler)
}

func extractSchedulerArgs(args ...interface{}) string {
	return args[0].(string)
}
