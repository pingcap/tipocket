package nemesis

import (
	"context"

	"github.com/pingcap/tipocket/pkg/cluster"

	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/util/net"
)

type kill struct{}

func (kill) Invoke(ctx context.Context, node cluster.Node, args ...string) error {
	db := core.GetDB(args[0])
	return db.Kill(ctx, node)
}

func (kill) Recover(ctx context.Context, node cluster.Node, args ...string) error {
	db := core.GetDB(args[0])
	return db.Start(ctx, node)
}

func (kill) Name() string {
	return "kill"
}

type drop struct {
	t net.IPTables
}

func (n drop) Invoke(ctx context.Context, node cluster.Node, args ...string) error {
	for _, dropNode := range args {
		if node.IP == dropNode {
			// Don't drop itself
			continue
		}

		if err := n.t.Drop(ctx, node.IP, dropNode); err != nil {
			return err
		}
	}
	return nil
}

func (n drop) Recover(ctx context.Context, node cluster.Node, args ...string) error {
	return n.t.Heal(ctx, node.IP)
}

func (drop) Name() string {
	return "drop"
}

func init() {
	core.RegisterNemesis(kill{})
	core.RegisterNemesis(drop{})
}
