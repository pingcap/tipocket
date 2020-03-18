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

package cdc

import (
	"context"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql" // mysql driver
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/util"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CdcOps knows how to operate TiDB CDC on k8s
type CdcOps struct {
	cli client.Client
}

func New(cli client.Client) *CdcOps {
	return &CdcOps{cli}
}

type CDCSpec struct {
	Namespace string
	Name      string
	Source    *v1alpha1.TidbCluster
	Replicas  int32
	Image     string
	Resources corev1.ResourceRequirements
}

type CDC struct {
	Sts    *appsv1.StatefulSet
	Source *v1alpha1.TidbCluster
}

type CDCJob struct {
	*CDC
	SinkURI  string
	StartTs  uint64
	TargetTs uint64
}

func (c *CdcOps) ApplyCDC(spec *CDCSpec) (*CDC, error) {
	if spec.Image == "" {
		spec.Image = fixture.Context.CDCImage
	}
	cc, err := c.renderCdc(spec)
	if err != nil {
		return nil, err
	}
	desiredSts := cc.Sts.DeepCopy()
	_, err = controllerutil.CreateOrUpdate(context.TODO(), c.cli, cc.Sts, func() error {
		cc.Sts.Spec.Template = desiredSts.Spec.Template
		cc.Sts.Spec.Replicas = desiredSts.Spec.Replicas
		cc.Sts.Spec.PodManagementPolicy = desiredSts.Spec.PodManagementPolicy
		return nil
	})
	return cc, err
}

func (c *CdcOps) DeleteCDC(cc *CDC) error {
	err := c.cli.Delete(context.TODO(), cc.Sts)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (c *CdcOps) StartJob(job *CDCJob, spec *CDCSpec) error {
	kjob, err := c.renderSyncJob(job, spec)
	if err != nil {
		return err
	}

	if err := c.cli.Create(context.TODO(), kjob); err != nil {
		return err
	}

	return nil
}

func (c CdcOps) renderSyncJob(job *CDCJob, spec *CDCSpec) (*batchv1.Job, error) {
	name := fmt.Sprintf("tipocket-cdc-%s", spec.Name)
	l := map[string]string{
		"app":      "tipocket-cdc",
		"instance": name,
		"source":   spec.Source.Name,
	}
	image := spec.Image
	if image == "" {
		image = fixture.Context.CDCImage
	}
	pdAddr := util.PDAddress(job.CDC.Source)

	cmds := []string{
		"/cdc",
		"cli",
		"changefeed",
		"create",
		fmt.Sprintf("--pd=%s", pdAddr),
		"--start-ts=0",
		fmt.Sprintf("--sink-uri=mysql://%s/", (strings.Split(job.SinkURI, "@")[0] + ":@" + strings.Split(strings.Split(job.SinkURI, "(")[1], ")")[0])),
	}

	syncJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: spec.Namespace,
			Labels:    l,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "cdc-cli",
							Image:           spec.Image,
							ImagePullPolicy: corev1.PullAlways,
							Command:         cmds,
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	return syncJob, nil
}

const (
	// EtcdKeyBase is the common prefix of the keys in CDC
	EtcdKeyBase = "/tidb/cdc"
)

func (c *CdcOps) StopJob(job *CDCJob) error {
	return fmt.Errorf("not implemented")
}

func (c *CdcOps) renderCdc(spec *CDCSpec) (*CDC, error) {
	name := fmt.Sprintf("tipocket-cdc-%s", spec.Name)
	l := map[string]string{
		"app":      "tipocket-cdc",
		"instance": name,
		"source":   spec.Source.Name,
	}
	image := spec.Image
	if image == "" {
		image = fixture.Context.CDCImage
	}
	pdAddr := util.PDAddress(spec.Source)
	cmds := []string{
		"/cdc",
		"server",
		fmt.Sprintf("--pd=%s", pdAddr),
		"--log-file=ticdc.log",
		"--status-addr=127.0.0.1:8301",
	}
	return &CDC{
		Sts: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: spec.Namespace,
				Labels:    l,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{MatchLabels: l},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: l},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:            "cdc",
							Command:         cmds,
							Image:           spec.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							// Resources:       operatorutil.ResourceRequirement(spec.Resources),
						}},
					},
				},
			},
		},
		Source: spec.Source,
	}, nil
}
