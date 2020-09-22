package nemesis

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	// TODO: manage chaos-operator dep in go mod
	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Chaos knows how to operate the chaos provided by pingcap/chaos-mesh
type Chaos struct {
	cli client.Client
}

// New will Create a chaos client.
func New(cli client.Client) *Chaos {
	return &Chaos{cli}
}

// ApplyIOChaos run an io-chaos.
func (c *Chaos) ApplyIOChaos(ioc *v1alpha1.IoChaos) error {
	desired := ioc.DeepCopy()
	_, err := controllerutil.CreateOrUpdate(context.TODO(), c.cli, ioc, func() error {
		ioc.Spec = desired.Spec
		return nil
	})
	return err
}

// CancelIOChaos cancel the io chaos.
func (c *Chaos) CancelIOChaos(ioc *v1alpha1.IoChaos) error {
	return c.cli.Delete(context.TODO(), ioc)
}

// ApplyNetChaos apply the chaos to cluster using Client.
func (c *Chaos) ApplyNetChaos(nc *v1alpha1.NetworkChaos) error {
	desired := nc.DeepCopy()
	_, err := controllerutil.CreateOrUpdate(context.TODO(), c.cli, nc, func() error {
		nc.Spec = desired.Spec
		return nil
	})
	return err
}

// CancelNetChaos apply the chaos to cluster using Client.
func (c *Chaos) CancelNetChaos(nc *v1alpha1.NetworkChaos) error {
	return c.cli.Delete(context.TODO(), nc)
}

// ApplyPodChaos apply the pod chaos to cluster using Client.
func (c *Chaos) ApplyPodChaos(ctx context.Context, pc *v1alpha1.PodChaos) error {
	desired := pc.DeepCopy()
	_, err := controllerutil.CreateOrUpdate(ctx, c.cli, pc, func() error {
		pc.Spec = desired.Spec
		return nil
	})
	return err
}

// CancelPodChaos Delete the pod chaos using Client.
func (c *Chaos) CancelPodChaos(ctx context.Context, pc *v1alpha1.PodChaos) error {
	return c.cli.Delete(ctx, pc)
}

// ApplyTimeChaos apply the pod chaos to cluster using Client.
func (c *Chaos) ApplyTimeChaos(ctx context.Context, pc *v1alpha1.TimeChaos) error {
	desired := pc.DeepCopy()
	_, err := controllerutil.CreateOrUpdate(ctx, c.cli, pc, func() error {
		pc.Spec = desired.Spec
		return nil
	})
	return err
}

// CancelTimeChaos Delete the pod chaos using Client.
func (c *Chaos) CancelTimeChaos(ctx context.Context, pc *v1alpha1.TimeChaos) error {
	return c.cli.Delete(ctx, pc)
}
