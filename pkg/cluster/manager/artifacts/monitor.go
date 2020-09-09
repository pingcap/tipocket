package artifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/juju/errors"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/pingcap/tipocket/pkg/cluster/manager/deploy"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
	"github.com/pingcap/tipocket/pkg/cluster/manager/util"
)

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
		fmt.Sprintf(`find provisioning -type f -exec gsed -i 's/%s\/dashboards/\/etc\/grafana\/dashboards/g' {} \;`,
			strings.ReplaceAll(deployDir, "/", `\/`)))
	if err != nil {
		return errors.Trace(err)
	}
	// normalize provisioning/datasources/datasource.yml
	_, err = util.Command(tmpDir, "bash", "-c",
		fmt.Sprintf(`find provisioning -type f -exec gsed -i 's/\/\/%s:/\/\/${PROM_ADDR}:/g' {} \;`, promServerHost))
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
