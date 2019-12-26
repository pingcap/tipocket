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

package etcd

import (
	"context"
	"fmt"

	"github.com/juju/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tipocket/test-infra/pkg/fixture"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ETCDOps struct {
	cli client.Client
}

func New(cli client.Client) *ETCDOps {
	return &ETCDOps{cli}
}

type ETCDSpec struct {
	Name      string
	Namespace string
	Version   string
	Replicas  int
	Resource  v1alpha1.Resources
	Storage   fixture.StorageType
}

type ETCD struct {
	Sts *appsv1.StatefulSet
	Svc *corev1.Service
}

func (e *ETCDOps) ApplyETCD(spec *ETCDSpec) (*ETCD, error) {
	toCreate, err := e.renderETCD(spec)
	if err != nil {
		return nil, err
	}

	desiredSts := toCreate.Sts.DeepCopy()
	_, err = controllerutil.CreateOrUpdate(context.TODO(), e.cli, toCreate.Sts, func() error {
		toCreate.Sts.Spec.Template = desiredSts.Spec.Template
		return nil
	})
	desiredSvc := toCreate.Svc.DeepCopy()
	_, err = controllerutil.CreateOrUpdate(context.TODO(), e.cli, toCreate.Svc, func() error {
		clusterIp := toCreate.Svc.Spec.ClusterIP
		toCreate.Svc.Spec = desiredSvc.Spec
		toCreate.Svc.Spec.ClusterIP = clusterIp
		return nil
	})

	return toCreate, nil
}

func (e *ETCDOps) DeleteETCD(etcd *ETCD) error {
	err := e.cli.Delete(context.TODO(), etcd.Sts)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	err = e.cli.Delete(context.TODO(), etcd.Svc)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}

func (e *ETCDOps) GetNodes(etcd *ETCD) ([]string, error) {
	pods := &corev1.PodList{}
	listOptions := &client.ListOptions{
		Namespace: etcd.Sts.Namespace,
	}
	if err := e.cli.List(context.TODO(), pods, listOptions); err != nil {
		return nil, nil
	}

	var nodes []string
	for _, pod := range pods.Items {
		nodes = append(nodes, pod.Status.PodIP)
	}

	return nodes, nil
}

func (e *ETCDOps) renderETCD(spec *ETCDSpec) (*ETCD, error) {
	name := fmt.Sprintf("e2e-etcd-%s", spec.Name)
	l := map[string]string{
		"app":      "e2e-etcd",
		"instance": name,
	}

	var q resource.Quantity
	var err error
	if spec.Resource.Requests == nil {
		spec.Resource.Requests = &v1alpha1.ResourceRequirement{
			Storage: "10G",
		}
	}

	size := spec.Resource.Requests.Storage
	q, err = resource.ParseQuantity(size)
	if err != nil {
		return nil, fmt.Errorf("cant' get storage size for mysql: %v", err)
	}

	return &ETCD{
		Svc: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: spec.Namespace,
				Labels:    l,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{{
					Name:       "etcd-server",
					Port:       2379,
					TargetPort: intstr.FromInt(2379),
					Protocol:   corev1.ProtocolTCP,
				}},
				Selector: l,
			},
		},
		Sts: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: spec.Namespace,
				Labels:    l,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointer.Int32Ptr(int32(spec.Replicas)),
				Selector: &metav1.LabelSelector{
					MatchLabels: l,
				},
				ServiceName: spec.Name,
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
					ObjectMeta: metav1.ObjectMeta{Name: "datadir"},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: pointer.StringPtr(fixture.StorageClass(spec.Storage)),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: q,
							},
						},
					},
				}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:   name,
						Labels: l,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:            "etcd",
							Image:           fmt.Sprintf("k8s.gcr.io/etcd-amd64:%s", spec.Version),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{Name: "peer", ContainerPort: 2380},
								{Name: "client", ContainerPort: 2379},
							},
							Resources: util.ResourceRequirement(spec.Resource),
							Env: []corev1.EnvVar{
								{Name: "ETCDCTL_API", Value: "3"},
								{Name: "INITIAL_CLUSTER_SIZE", Value: fmt.Sprintf("%d", spec.Replicas)},
								{Name: "SET_NAME", Value: spec.Name},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "datadir", MountPath: "/var/run/etcd"},
							},
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/sh",
											"-ec",
											`
EPS=""
for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
	EPS="${EPS}${EPS:+,}http://${SET_NAME}-${i}.${SET_NAME}:2379"
done

HOSTNAME=$(hostname)

AUTH_OPTIONS=""                  

member_hash() {
	etcdctl $AUTH_OPTIONS member list | grep http://${HOSTNAME}.${SET_NAME}:2380 | cut -d':' -f1 | cut -d'[' -f1
}

SET_ID=${HOSTNAME##*[^0-9]}

if [ "${SET_ID}" -ge ${INITIAL_CLUSTER_SIZE} ]; then
	echo "Removing ${HOSTNAME} from etcd cluster"
	ETCDCTL_ENDPOINT=${EPS} etcdctl $AUTH_OPTIONS member remove $(member_hash)
	if [ $? -eq 0 ]; then
		# Remove everything otherwise the cluster will no longer scale-up
		rm -rf /var/run/etcd/*
	fi
fi
`,
										},
									},
								},
							},
							Command: []string{
								"/bin/sh",
								"-ec",
								`
HOSTNAME=$(hostname)
AUTH_OPTIONS=""

# store member id into PVC for later member replacement
collect_member() {                
	while ! etcdctl $AUTH_OPTIONS member list &>/dev/null; do sleep 1; done
	etcdctl $AUTH_OPTIONS member list | grep http://${HOSTNAME}.${SET_NAME}:2380 | cut -d':' -f1 | cut -d'[' -f1 > /var/run/etcd/member_id                
	exit 0
}

eps() {
	EPS=""
	for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
		EPS="${EPS}${EPS:+,}http://${SET_NAME}-${i}.${SET_NAME}:2379"
	done
	echo ${EPS}
}

member_hash() {
	etcdctl $AUTH_OPTIONS member list | grephttp://${HOSTNAME}.${SET_NAME}:2380 | cut -d':' -f1 | cut -d'[' -f1
}

# we should wait for other pods to be up before trying to join
# otherwise we got "no such host" errors when trying to resolve other members
for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
	while true; do
		echo "Waiting for ${SET_NAME}-${i}.${SET_NAME} to come up"
		ping -W 1 -c 1 ${SET_NAME}-${i}.${SET_NAME} > /dev/null && break
		sleep 1s
	done                
done
            
# re-joining after failure?
if [ -e /var/run/etcd/default.etcd ]; then
	echo "Re-joining etcd member"
	member_id=$(cat /var/run/etcd/member_id)
	# re-join member
	ETCDCTL_ENDPOINT=$(eps) etcdctl $AUTH_OPTIONS member update ${member_id} http://${HOSTNAME}.${SET_NAME}:2380 | true
	exec etcd --name ${HOSTNAME} \
		--listen-peer-urls http://0.0.0.0:2380 \
		--listen-client-urls http://0.0.0.0:2379\
		--advertise-client-urls http://${HOSTNAME}.${SET_NAME}:2379 \
		--data-dir /var/run/etcd/default.etcd
                    
fi

# etcd-SET_ID
SET_ID=${HOSTNAME##*[^0-9]}

# adding a new member to existing cluster (assuming all initial pods are available)
if [ "${SET_ID}" -ge ${INITIAL_CLUSTER_SIZE} ]; then
	export ETCDCTL_ENDPOINT=$(eps)

	# member already added?
	MEMBER_HASH=$(member_hash)
	if [ -n "${MEMBER_HASH}" ]; then
		# the member hash exists but for some reason etcd failed
		# as the datadir has not be created, we can remove the member
		# and retrieve new hash
		etcdctl $AUTH_OPTIONS member remove ${MEMBER_HASH}
	fi

	echo "Adding new member"
	etcdctl $AUTH_OPTIONS member add ${HOSTNAME} http://${HOSTNAME}.${SET_NAME}:2380 | grep "^ETCD_" > /var/run/etcd/new_member_envs

	if [ $? -ne 0 ]; then
		echo "Exiting"
		rm -f /var/run/etcd/new_member_envs
		exit 1
	fi

	cat /var/run/etcd/new_member_envs
	source /var/run/etcd/new_member_envs

	collect_member &

	exec etcd --name ${HOSTNAME} \
		--listen-peer-urls http://0.0.0.0:2380 \
 		--listen-client-urls http://0.0.0.0:2379 \
 		--advertise-client-urls http://${HOSTNAME}.${SET_NAME}:2379 \
 		--data-dir /var/run/etcd/default.etcd \
 		--initial-advertise-peer-urls http://${HOSTNAME}.${SET_NAME}:2380 \
 		--initial-cluster ${ETCD_INITIAL_CLUSTER} \
 		--initial-cluster-state ${ETCD_INITIAL_CLUSTER_STATE}
                    
fi

PEERS=""
for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
	PEERS="${PEERS}${PEERS:+,}${SET_NAME}-${i}=http://${SET_NAME}-${i}.${SET_NAME}:2380"
done

collect_member &

# join member
exec etcd --name ${HOSTNAME} \
	--initial-advertise-peer-urls http://${HOSTNAME}.${SET_NAME}:2380 \
 	--listen-peer-urls http://0.0.0.0:2380 \
 	--listen-client-urls http://0.0.0.0:2379 \
 	--advertise-client-urls http://${HOSTNAME}.${SET_NAME}:2379 \
 	--initial-cluster-token etcd-cluster-1 \
 	--initial-cluster ${PEERS} \
 	--initial-cluster-state new \
 	--data-dir /var/run/etcd/default.etcd
`,
							},
						}},
					},
				},
			},
		},
	}, nil
}
