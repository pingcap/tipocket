package artifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	testUtil "github.com/pingcap/tipocket/pkg/test-infra/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tipocket/pkg/cluster/manager/deploy"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
	"github.com/pingcap/tipocket/pkg/cluster/manager/util"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
)

const namespace = "tipocket"

func ArchiveMonitorData(uuid string, topos *deploy.Topology) (err error) {
	var (
		promHost    string
		grafanaHost string
		promTopo    *types.ClusterRequestTopology
		grafanaTopo *types.ClusterRequestTopology
	)
	s3Client, err := NewS3Client()
	if err != nil {
		return errors.Trace(err)
	}
	if err != nil {
		return errors.Trace(err)
	}
	if len(topos.PrometheusServers) == 0 {
		return errors.Trace(errors.NotFoundf("prometheus server"))
	}
	if len(topos.GrafanaServers) == 0 {
		return errors.Trace(errors.NotFoundf("grafana server"))
	}
	for host, topo := range topos.PrometheusServers {
		promHost = host
		promTopo = topo
	}
	for host, topo := range topos.PrometheusServers {
		grafanaHost = host
		grafanaTopo = topo
	}
	if err := archiveProm(s3Client, uuid, promHost, promTopo); err != nil {
		return errors.Trace(err)
	}
	if err := archiveGrafana(s3Client, uuid, promHost, grafanaHost, grafanaTopo); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func archiveProm(s3Client *S3Client, uuid string, promServerHost string, promServerTopo *types.ClusterRequestTopology) error {
	type Snapshot struct {
		Name  string `json:"name"`
		Error string `json:"error"`
	}
	var snapshot Snapshot
	tmpDir, err := ioutil.TempDir("", "artifacts-prom")
	if err != nil {
		return err
	}
	// http://172.16.5.110:9090/api/v2/admin/tsdb/snapshot
	resp, err := http.Post(fmt.Sprintf("http://%s:9090/api/v2/admin/tsdb/snapshot", promServerHost), "application/json", nil)
	if err != nil {
		return errors.Trace(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Trace(err)
	}
	if err := json.Unmarshal(body, &snapshot); err != nil {
		return errors.Trace(err)
	}
	if resp.StatusCode != 200 {
		return errors.Trace(errors.New(snapshot.Error))
	}
	// FIXME(@mahjonp): should use non-prompt to avoid the command hangs
	output, err := util.Command(tmpDir,
		"rsync",
		"-avz",
		fmt.Sprintf("tidb@%s:%s/snapshots/%s", promServerHost, deploy.BuildNormalPrometheusDataDir(promServerTopo), snapshot.Name), ".")
	if err != nil {
		return err
	}
	zap.L().Debug("rsync success", zap.String("dir", tmpDir), zap.String("output", output))
	fileName := "prometheus.tar.gz"
	_, err = util.Command(tmpDir, "tar", "zcf", fileName, snapshot.Name)
	if err != nil {
		return errors.Trace(err)
	}
	_, err = s3Client.FPutObject(context.Background(), "artifacts", fmt.Sprintf("%s/%s", uuid, fileName), path.Join(tmpDir, fileName), minio.PutObjectOptions{})
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func archiveGrafana(s3Client *S3Client, uuid string, promServerHost string, grafanaServerHost string, grafanaTopo *types.ClusterRequestTopology) error {
	tmpDir, err := ioutil.TempDir("", "artifacts-prom")
	if err != nil {
		return err
	}
	deployDir := deploy.BuildNormalGrafanaDeployDir(grafanaTopo)
	output, err := util.Command(tmpDir,
		"bash",
		"-c",
		fmt.Sprintf(`rsync -avz tidb@%s:'%s/provisioning %s/dashboards' .`, grafanaServerHost, deployDir, deployDir))
	if err != nil {
		return err
	}
	zap.L().Debug("rsync success", zap.String("dir", tmpDir), zap.String("output", output))

	// normalize provisioning/dashboards/dashboard.yml
	_, err = util.Command(tmpDir, "bash", "-c",
		fmt.Sprintf(`find provisioning -type f -exec sed -i 's/%s\/dashboards/\/etc\/grafana\/dashboards/g' {} \;`,
			strings.ReplaceAll(deployDir, "/", `\/`)))
	if err != nil {
		return errors.Trace(err)
	}
	// normalize provisioning/datasources/datasource.yml
	_, err = util.Command(tmpDir, "bash", "-c",
		fmt.Sprintf(`find provisioning -type f -exec sed -i 's/\/\/%s:/\/\/${PROM_ADDR}:/g' {} \;`, promServerHost))
	if err != nil {
		return errors.Trace(err)
	}
	fileName := "grafana.tar.gz"
	_, err = util.Command(tmpDir, "tar", "zcf", fileName, "dashboards", "provisioning")
	if err != nil {
		return errors.Trace(err)
	}
	_, err = s3Client.FPutObject(context.Background(), "artifacts", fmt.Sprintf("%s/%s", uuid, fileName), path.Join(tmpDir, fileName), minio.PutObjectOptions{})
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Rebuild monitoring on K8s cluster
func RebuildMonitoringOnK8s(uuid string) (err error) {
	err = rebuildProm(uuid)
	if err != nil {
		return err
	}
	return rebuildGrafana(uuid)
}

func rebuildProm(uuid string) (err error) {
	monitoringPodName := fmt.Sprintf("monitoring-%s", uuid)
	monitoringClaimName := fmt.Sprintf("monitoring-claim-%s", uuid)
	monitoringService := fmt.Sprintf("monitoring-service-%s", uuid)
	volumeClaim := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      monitoringClaimName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
		Status: corev1.PersistentVolumeClaimStatus{},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      monitoringPodName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":  "monitoring",
				"uuid": uuid,
			},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "monitoring-storage",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: monitoringClaimName,
						},
					},
				},
			},
			InitContainers: []corev1.Container{{
				Name:  "restore-data",
				Image: "minio/mc",
				Command: []string{
					"/bin/sh",
					"-c",
					fmt.Sprintf(`set -euo pipefail
cd prometheus
mc alias set minio http://%s %s %s
mc cp minio/artifacts/%s/prometheus.tar.gz .
tar xf prometheus.tar.gz --strip-components 1
chown -R nobody:nobody .
`,
						util.S3Endpoint, util.AwsAccessKeyID, util.AwsSecretAccessKey, uuid),
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "monitoring-storage",
						MountPath: "/prometheus",
					},
				},
			}},
			Containers: []corev1.Container{{
				Name:  "prometheus",
				Image: "prom/prometheus",
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: 9090,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "monitoring-storage",
						MountPath: "/prometheus",
					},
				},
			}},
		},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      monitoringService,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":  "monitoring",
				"uuid": uuid,
			},
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     9090,
				},
			},
		},
	}
	for _, obj := range []runtime.Object{volumeClaim, pod, service} {
		err = testUtil.ApplyObject(tests.TestClient.Cli, obj)
		if err != nil {
			return err
		}
	}
	return wait.PollImmediate(10*time.Second, 5*time.Minute, func() (done bool, err error) {
		local := pod.DeepCopy()
		key, err := client.ObjectKeyFromObject(local)
		if err := tests.TestClient.Cli.Get(context.TODO(), key, local); err != nil {
			return false, err
		}
		return local.Status.Phase == corev1.PodRunning, nil
	})
}

func rebuildGrafana(uuid string) (err error) {
	grafanaPodName := fmt.Sprintf("grafana-%s", uuid)
	grafanaService := fmt.Sprintf("grafana-service-%s", uuid)
	monitoringService := fmt.Sprintf("monitoring-service-%s", uuid)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      grafanaPodName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":  "grafana",
				"uuid": uuid,
			},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "grafana-configuration",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			InitContainers: []corev1.Container{{
				Name:  "restore-data",
				Image: "minio/mc",
				Command: []string{
					"/bin/sh",
					"-c",
					fmt.Sprintf(`set -uo pipefail
cd /etc/grafana
mc alias set minio http://%s %s %s
mc cp minio/artifacts/%s/grafana.tar.gz .
tar xf grafana.tar.gz
find . -type f -exec sed -i "s/\${PROM_ADDR}/%s.%s.svc/g" {} \;
touch grafana.ini
chown -R 472:472 /etc/grafana
ls -althr
`,
						util.S3Endpoint, util.AwsAccessKeyID, util.AwsSecretAccessKey, uuid, monitoringService, namespace),
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "grafana-configuration",
						MountPath: "/etc/grafana",
					},
				},
			}},
			Containers: []corev1.Container{{
				Name:  "grafana",
				Image: "grafana/grafana",
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: 3000,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "grafana-configuration",
						MountPath: "/etc/grafana",
					},
				},
			}},
		},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      grafanaService,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":  "grafana",
				"uuid": uuid,
			},
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     3000,
				},
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}
	for _, obj := range []runtime.Object{pod, service} {
		err = testUtil.ApplyObject(tests.TestClient.Cli, obj)
		if err != nil {
			return err
		}
	}
	return wait.PollImmediate(10*time.Second, 5*time.Minute, func() (done bool, err error) {
		local := pod.DeepCopy()
		key, err := client.ObjectKeyFromObject(local)
		if err := tests.TestClient.Cli.Get(context.TODO(), key, local); err != nil {
			return false, err
		}
		return local.Status.Phase == corev1.PodRunning, nil
	})
}
