// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package chaos

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	// TODO: manage chaos-operaot dep in go mod
	"github.com/pingcap/chaos-mesh/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Chaos knows how to operate the chaos provided by pingcap/chaos-mesh
type Chaos struct {
	cli client.Client
}

func New(cli client.Client) *Chaos {
	return &Chaos{cli}
}

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

func (c *Chaos) CancelPodChaos(pc *v1alpha1.PodChaos) error {
	return c.cli.Delete(context.TODO(), pc)
}
