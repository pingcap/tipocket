package deploy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
	"github.com/pingcap/tipocket/pkg/cluster/manager/util"
)

// Topology ...
type Topology struct {
	Config            string
	PDServers         map[string]*types.ClusterRequestTopology
	TiKVServers       map[string]*types.ClusterRequestTopology
	TiDBServers       map[string]*types.ClusterRequestTopology
	PrometheusServers map[string]*types.ClusterRequestTopology
	GrafanaServers    map[string]*types.ClusterRequestTopology
}

// TryDeployCluster ...
func TryDeployCluster(name string,
	resources []*types.Resource,
	rris []*types.ResourceRequestItem,
	cr *types.ClusterRequest,
	crts []*types.ClusterRequestTopology) (*Topology, error) {
	topo := &Topology{
		Config:            cr.Config,
		PDServers:         make(map[string]*types.ClusterRequestTopology),
		TiKVServers:       make(map[string]*types.ClusterRequestTopology),
		TiDBServers:       make(map[string]*types.ClusterRequestTopology),
		PrometheusServers: make(map[string]*types.ClusterRequestTopology),
		GrafanaServers:    make(map[string]*types.ClusterRequestTopology),
	}
	id2Resource := make(map[uint]*types.Resource)
	rriItemID2ResourceID := make(map[uint]uint)
	// resource request item id -> resource
	for _, re := range resources {
		id2Resource[re.ID] = re
	}
	// resource request item item_id -> resource request item id
	for _, rri := range rris {
		rriItemID2ResourceID[rri.ItemID] = rri.RID
	}
	for idx, crt := range crts {
		ip := id2Resource[rriItemID2ResourceID[crt.RRIItemID]].IP
		switch strings.ToLower(crt.Component) {
		case "tidb":
			topo.TiDBServers[ip] = crts[idx]
		case "tikv":
			topo.TiKVServers[ip] = crts[idx]
		case "pd":
			topo.PDServers[ip] = crts[idx]
		case "prometheus":
			topo.PrometheusServers[ip] = crts[idx]
		case "grafana":
			topo.GrafanaServers[ip] = crts[idx]
		default:
			return nil, fmt.Errorf("unknown component type %s", crt.Component)
		}
	}

	yaml := buildTopologyYaml(topo)
	if err := deployCluster(topo, yaml, name, cr); err != nil {
		return nil, errors.Trace(err)
	}
	if err := startCluster(name); err != nil {
		return nil, errors.Trace(err)
	}
	// FIXME(mahjonp): replace it with heath check
	// wait a moment for tidb providing service
	time.Sleep(20 * time.Second)
	return topo, nil
}

// TryScaleOut ...
func TryScaleOut(name string, r *types.Resource, component string) error {
	host := r.IP
	// FIXME @mahjonp
	deployPath := "/data1"
	var tpl string
	switch component {
	case "tikv":
		tpl = fmt.Sprintf(`
tikv_servers:
- host: %s
  port: 20160
  status_port: 20180
  deploy_dir: %s
  data_dir: %s
`, host, TiKVDeployPath(deployPath), TiKVDataPath(deployPath))
	default:
		return fmt.Errorf("unsupport component type %s now", component)
	}

	file, err := ioutil.TempFile("", "scale-out")
	if err != nil {
		return errors.Trace(err)
	}
	defer file.Close()
	if _, err := file.WriteString(tpl); err != nil {
		return errors.Trace(err)
	}

	output, err := util.Command("", "tiup", "cluster", "scale-out", "-y", name, file.Name())
	if err != nil {
		return fmt.Errorf("scale-out cluster failed, err: %v, output: %s", err, output)
	}
	return nil
}

// TryScaleIn ...
func TryScaleIn(name string, r *types.Resource, component string) error {
	host := r.IP
	var node string
	switch component {
	case "tikv":
		// FIXME @mahjonp
		node = fmt.Sprintf("%s:20160", host)
	default:
		return fmt.Errorf("unsupport component type %s now", component)
	}
	output, err := util.Command("", "tiup", "cluster", "scale-in", "-y", name, "--node", node)
	if err != nil {
		return fmt.Errorf("scale-in cluster failed, err: %v, output: %s", err, output)
	}
	return nil
}

// StartCluster calls tiup cluster start xxx
func StartCluster(name string) error {
	output, err := util.Command("", "tiup", "cluster", "start", name)
	if err != nil {
		return fmt.Errorf("start cluster failed, err: %v, output: %s", err, output)
	}
	return nil
}

// StopCluster calls tiup cluster stop xxx
func StopCluster(name string) error {
	output, err := util.Command("", "tiup", "cluster", "stop", name)
	if err != nil {
		return fmt.Errorf("stop cluster failed, err: %v, output: %s", err, output)
	}
	return nil
}

// DestroyCluster ...
func DestroyCluster(name string) error {
	output, err := util.Command("", "tiup", "cluster", "destroy", name, "-y")
	if err != nil {
		return fmt.Errorf("destroy cluster failed, err: %v, output: %s", err, output)
	}
	return nil
}

// CleanClusterData ...
func CleanClusterData(name string) error {
	output, err := util.Command("", "tiup", "cluster", "clean", name, "--data", "--ignore-role", "prometheus", "-y")
	if err != nil {
		return fmt.Errorf("clean cluster failed, err: %v, output: %s", err, output)
	}
	return nil
}

func deployCluster(topo *Topology, yaml, name string, cr *types.ClusterRequest) error {
	file, err := ioutil.TempFile("", "cluster")
	if err != nil {
		return errors.Trace(err)
	}
	defer file.Close()
	file.WriteString(yaml)

	_, err = util.Command("", "tiup", "cluster", "deploy", "-y", name, cr.Version, file.Name())
	if err != nil {
		return fmt.Errorf("deploy cluster failed, err: %s", err)
	}
	_, err = util.Command("", "tiup", "cluster", "start", name)
	if err != nil {
		return fmt.Errorf("start cluster failed, err: %s", err)
	}
	return patchCluster(topo, name, cr)
}

func patchCluster(topo *Topology, name string, cr *types.ClusterRequest) error {
	var patchComponent = []struct {
		component   string
		downloadURL string
	}{
		{
			component:   "tidb-server",
			downloadURL: cr.TiDBVersion,
		},
		{
			component:   "tikv-server",
			downloadURL: cr.TiKVVersion,
		},
		{
			component:   "pd-server",
			downloadURL: cr.PDVersion,
		},
	}
	needPatch := false
	for _, component := range patchComponent {
		if component.component == "tidb-server" && len(topo.TiDBServers) == 0 {
			continue
		}
		if component.component == "tikv-server" && len(topo.TiKVServers) == 0 {
			continue
		}
		if component.component == "pd-server" && len(topo.PDServers) == 0 {
			continue
		}
		if len(component.downloadURL) != 0 {
			needPatch = true
			break
		}
	}
	if !needPatch {
		return nil
	}
	dir, err := ioutil.TempDir("", "components")
	if err != nil {
		return errors.Trace(err)
	}
	var components []string
	for _, component := range patchComponent {
		if component.component == "tidb-server" && len(topo.TiDBServers) == 0 {
			continue
		}
		if component.component == "tikv-server" && len(topo.TiKVServers) == 0 {
			continue
		}
		if component.component == "pd-server" && len(topo.PDServers) == 0 {
			continue
		}
		if len(component.downloadURL) != 0 {
			components = append(components, component.component)
			filePath, err := util.Wget(component.downloadURL, dir)
			if err != nil {
				return errors.Trace(err)
			}
			if err := util.Unzip(filePath, dir); err != nil {
				return errors.Trace(err)
			}
			if _, err := util.Command(dir, "mv", fmt.Sprintf("bin/%s", component.component), component.component); err != nil {
				return errors.Trace(err)
			}
		}
	}
	args := append([]string{"zcf", "patch.tar.gz"}, components...)
	if _, err := util.Command(dir, "tar", args...); err != nil {
		return errors.Trace(err)
	}

	for _, component := range components {
		if _, err := util.Command(dir, "tiup", "cluster", "patch", name, "patch.tar.gz", "-R", strings.Split(component, "-server")[0]); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func startCluster(name string) error {
	output, err := util.Command("", "tiup", "cluster", "start", name)
	if err != nil {
		return fmt.Errorf("start cluster failed, err: %v, output: %s", err, output)
	}
	return nil
}

func buildTopologyYaml(t *Topology) string {
	var topo bytes.Buffer
	topo.WriteString(`global:
  user: "tidb"
  ssh_port: 22
  arch: "amd64"

` + t.Config)

	topo.WriteString(`

pd_servers:`)
	for host, config := range t.PDServers {
		topo.WriteString(fmt.Sprintf(`
  - host: %s
    deploy_dir: "%s"
    data_dir: "%s"`, host, PDDeployPath(config.DeployPath), PDDataPath(config.DeployPath)))
	}

	if len(t.TiDBServers) != 0 {
		topo.WriteString(`

tidb_servers:`)
		for host, config := range t.TiDBServers {
			topo.WriteString(fmt.Sprintf(`
  - host: %s
    deploy_dir: "%s"`, host, TiDBDeployPath(config.DeployPath)))
		}
	}

	if len(t.TiKVServers) != 0 {
		topo.WriteString(`

tikv_servers:`)
		for host, config := range t.TiKVServers {
			topo.WriteString(fmt.Sprintf(`
  - host: %s
    deploy_dir: "%s"
    data_dir: "%s"`, host, TiKVDeployPath(config.DeployPath), TiKVDataPath(config.DeployPath)))
		}
	}

	topo.WriteString(`

monitoring_servers:`)
	for host, config := range t.PrometheusServers {
		topo.WriteString(fmt.Sprintf(`
  - host: %s
    deploy_dir: "%s"
    data_dir: "%s"`, host, BuildNormalPrometheusDeployDir(config), BuildNormalPrometheusDataDir(config)))
	}

	topo.WriteString(`

grafana_servers:`)
	for host, config := range t.GrafanaServers {
		topo.WriteString(fmt.Sprintf(`
  - host: %s
    deploy_dir: "%s"`, host, BuildNormalGrafanaDeployDir(config)))
	}
	return topo.String()
}

// TiDBDeployPath builds tidb deploy path
func TiDBDeployPath(volumePath string) string {
	return path.Join(volumePath, "deploy/tidb-4000")
}

// TiKVDeployPath builds tidb deploy path
func TiKVDeployPath(volumePath string) string {
	return path.Join(volumePath, "deploy/tikv-20160")
}

// TiKVDataPath builds tidb deploy path
func TiKVDataPath(volumePath string) string {
	return path.Join(volumePath, "data/tikv-20160")
}

// PDDeployPath builds tidb deploy path
func PDDeployPath(volumePath string) string {
	return path.Join(volumePath, "deploy/pd-2379")
}

// PDDataPath builds tidb deploy path
func PDDataPath(volumePath string) string {
	return path.Join(volumePath, "data/pd-2379")
}

// BuildNormalPrometheusDeployDir builds $root/deploy/prometheus-8249 format path
func BuildNormalPrometheusDeployDir(topology *types.ClusterRequestTopology) string {
	return fmt.Sprintf("%s/deploy/prometheus-8249", topology.DeployPath)
}

// BuildNormalPrometheusDataDir builds $root/data/prometheus-8249 format path
func BuildNormalPrometheusDataDir(topology *types.ClusterRequestTopology) string {
	return fmt.Sprintf("%s/data/prometheus-8249", topology.DeployPath)
}

// BuildNormalGrafanaDeployDir builds $root/data/grafana-3000 format path
func BuildNormalGrafanaDeployDir(topology *types.ClusterRequestTopology) string {
	return fmt.Sprintf("%s/deploy/grafana-3000", topology.DeployPath)
}
