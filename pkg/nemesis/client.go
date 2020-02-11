package nemesis

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	// TODO: manage chaos-operator dep in go mod
	"github.com/pingcap/chaos-mesh/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Chaos knows how to operate the chaos provided by pingcap/chaos-mesh
type Chaos struct {
	cli client.Client
}

// Create a chaos client
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

func (c *Chaos) CancelIOChaos(ioc *v1alpha1.IoChaos) error {
	return c.cli.Delete(context.TODO(), ioc)
}

func (c *Chaos) ApplyNetChaos(nc *v1alpha1.NetworkChaos) error {
	desired := nc.DeepCopy()
	_, err := controllerutil.CreateOrUpdate(context.TODO(), c.cli, nc, func() error {
		nc.Spec = desired.Spec
		return nil
	})
	return err
}

func (c *Chaos) CancelNetChaos(nc *v1alpha1.NetworkChaos) error {
	return c.cli.Delete(context.TODO(), nc)
}

func (c *Chaos) ApplyPodChaos(pc *v1alpha1.PodChaos) error {
	desired := pc.DeepCopy()
	_, err := controllerutil.CreateOrUpdate(context.TODO(), c.cli, pc, func() error {
		pc.Spec = desired.Spec
		return nil
	})
	return err
}

// Delete the pod chaos using Client.
func (c *Chaos) CancelPodChaos(pc *v1alpha1.PodChaos) error {
	return c.cli.Delete(context.TODO(), pc)
}
