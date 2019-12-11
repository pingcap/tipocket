package tidb

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tipocket/test-infra/pkg/fixture"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TiDBClusterRecommendation struct {
	*v1alpha1.TidbCluster
}

func (t *TiDBClusterRecommendation) Make() *v1alpha1.TidbCluster {
	return t.TidbCluster
}

func (t *TiDBClusterRecommendation) EnablePump(replicas int32) *TiDBClusterRecommendation {
	if t.Spec.Pump == nil {
		t.Spec.Pump = &v1alpha1.PumpSpec{
			Replicas: replicas,
			ComponentSpec: v1alpha1.ComponentSpec{
				BaseImage: "pingcap/tidb-binlog",
			},
			Resources: fixture.Medium,
		}
	}
	return t
}

func (t *TiDBClusterRecommendation) PDReplicas(replicas int32) *TiDBClusterRecommendation {
	t.TidbCluster.Spec.PD.Replicas = replicas
	return t
}

func (t *TiDBClusterRecommendation) TiKVReplicas(replicas int32) *TiDBClusterRecommendation {
	t.TidbCluster.Spec.TiKV.Replicas = replicas
	return t
}

func (t *TiDBClusterRecommendation) TiDBReplicas(replicas int32) *TiDBClusterRecommendation {
	t.TidbCluster.Spec.TiDB.Replicas = replicas
	return t
}

// tidb-operator do not have sane defaults yet, do a recommendation
func RecommendedTiDBCluster(ns string, name string) *TiDBClusterRecommendation {
	return &TiDBClusterRecommendation{
		TidbCluster: &v1alpha1.TidbCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels: map[string]string{
					"app":      "e2e-tidbcluster",
					"instance": "name",
				},
			},
			Spec: v1alpha1.TidbClusterSpec{
				Version:         fixture.E2eContext.TiDBVersion,
				PVReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
				EnablePVReclaim: true,
				PD: v1alpha1.PDSpec{
					Replicas:  3,
					Resources: fixture.Small,
					ComponentSpec: v1alpha1.ComponentSpec{
						BaseImage: fmt.Sprintf("%s/pd", fixture.E2eContext.DockerRepository),
					},
				},
				TiKV: v1alpha1.TiKVSpec{
					Replicas:  3,
					Resources: fixture.Medium,
					ComponentSpec: v1alpha1.ComponentSpec{
						BaseImage: fmt.Sprintf("%s/tikv", fixture.E2eContext.DockerRepository),
					},
				},
				TiDB: v1alpha1.TiDBSpec{
					Replicas:  2,
					Resources: fixture.Medium,
					Service: &v1alpha1.TiDBServiceSpec{
						ServiceSpec: v1alpha1.ServiceSpec{
							Type: corev1.ServiceTypeClusterIP,
						},
					},
					ComponentSpec: v1alpha1.ComponentSpec{
						BaseImage: fmt.Sprintf("%s/tidb", fixture.E2eContext.DockerRepository),
					},
				},
			},
		},
	}
}
