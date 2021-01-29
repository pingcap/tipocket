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

package nemesis

import (
	"context"
	"log"
	"math/rand"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/tidb-operator/apis/pingcap/v1alpha1"
)

type scalingGenerator struct {
	name string
}

// NewScalingGenerator creates a generator.
func NewScalingGenerator(name string) core.NemesisGenerator {
	return scalingGenerator{name: name}
}

func (s scalingGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	ops := make([]*core.NemesisOperation, 1)
	ops[0] = &core.NemesisOperation{
		Type:        core.Scaling,
		Node:        &nodes[0],
		InvokeArgs:  nil,
		RecoverArgs: nil,
		RunTime:     time.Second * time.Duration(rand.Intn(120)+60),
	}
	return ops
}

func (s scalingGenerator) Name() string {
	return s.name
}

type scaling struct {
	k8sNemesisClient
}

func (s scaling) Invoke(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	log.Printf("apply nemesis scaling on ns %s", node.Namespace)
	return s.scaleCluster(ctx, node.Namespace, node.Namespace, 2)
}

func (s scaling) Recover(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	log.Printf("unapply nemesis scaling on ns %s", node.Namespace)
	return s.scaleCluster(ctx, node.Namespace, node.Namespace, -2)
}

func (s scaling) scaleCluster(ctx context.Context, ns, name string, n int32) error {
	var tc v1alpha1.TidbCluster
	err := wait.PollImmediate(5*time.Second, time.Minute*time.Duration(5), func() (bool, error) {
		key := types.NamespacedName{
			Namespace: ns,
			Name:      name,
		}
		if err := s.cli.cli.Get(context.TODO(), key, &tc); err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	_, err = controllerutil.CreateOrUpdate(context.TODO(), s.cli.cli, &tc, func() error {
		tc.Spec.TiDB.Replicas += n
		tc.Spec.PD.Replicas += n
		tc.Spec.TiKV.Replicas += n
		log.Printf("Scale TiDB's nodes to %d PD's nodes to %d TiKV's nodes to %d", tc.Spec.TiDB.Replicas, tc.Spec.PD.Replicas, tc.Spec.TiKV.Replicas)
		return nil
	})
	return err
}

func (scaling) Name() string {
	return string(core.Scaling)
}
