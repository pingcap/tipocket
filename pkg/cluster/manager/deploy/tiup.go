package deploy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

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
	if err := deployCluster(yaml, name, cr); err != nil {
		return nil, errors.Trace(err)
	}
	if err := startCluster(name); err != nil {
		return nil, errors.Trace(err)
	}
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
  deploy_dir: %s/deploy/deploy/tikv-20160
  data_dir: %s/data/tikv-20160
`, host, deployPath, deployPath)
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

func TryStopCluster(name string) error {
	output, err := util.Command("", "tiup", "cluster", "stop", name)
	if err != nil {
		return fmt.Errorf("stop cluster failed, err: %v, output: %s", err, output)
	}
	return nil
}

// TryDestroyCluster ...
func TryDestroyCluster(name string) error {
	output, err := util.Command("", "tiup", "cluster", "destroy", name, "-y")
	if err != nil {
		return fmt.Errorf("destroy cluster failed, err: %v, output: %s", err, output)
	}
	return nil
}

func deployCluster(yaml, name string, cr *types.ClusterRequest) error {
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
	return patchCluster(name, cr)
}

func patchCluster(name string, cr *types.ClusterRequest) error {
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
    deploy_dir: "%s/deploy/pd-2379"
    data_dir: "%s/data/pd-2379"`, host, config.DeployPath, config.DeployPath))
	}

	topo.WriteString(`

tidb_servers:`)
	for host, config := range t.TiDBServers {
		topo.WriteString(fmt.Sprintf(`
  - host: %s
    deploy_dir: "%s/deploy/tidb-4000"`, host, config.DeployPath))
	}

	topo.WriteString(`

tikv_servers:`)
	for host, config := range t.TiKVServers {
		topo.WriteString(fmt.Sprintf(`
  - host: %s
    deploy_dir: "%s/deploy/tikv-20160"
    data_dir: "%s/data/tikv-20160"`, host, config.DeployPath, config.DeployPath))
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
