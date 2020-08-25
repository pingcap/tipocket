package deploy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/util"

	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

type Topology struct {
	Config            string
	PDServers         map[string]*types.ClusterRequestTopology
	TiKVServers       map[string]*types.ClusterRequestTopology
	TiDBServers       map[string]*types.ClusterRequestTopology
	MonitoringServers map[string]*types.ClusterRequestTopology
	GrafanaServers    map[string]*types.ClusterRequestTopology
}

func TryDeployCluster(name string,
	resources []types.Resource,
	rris []*types.ResourceRequestItem,
	cr *types.ClusterRequest,
	crts []*types.ClusterRequestTopology) error {
	topo := &Topology{
		Config:            cr.Config,
		PDServers:         make(map[string]*types.ClusterRequestTopology),
		TiKVServers:       make(map[string]*types.ClusterRequestTopology),
		TiDBServers:       make(map[string]*types.ClusterRequestTopology),
		MonitoringServers: make(map[string]*types.ClusterRequestTopology),
		GrafanaServers:    make(map[string]*types.ClusterRequestTopology),
	}
	rriID2Resource := make(map[uint]types.Resource)
	rriItemId2RriID := make(map[uint]uint)
	// resource request item id -> resource
	for _, re := range resources {
		rriID2Resource[re.RRIID] = re
	}
	// resource request item item_id ->  resource request item id
	for _, rri := range rris {
		rriItemId2RriID[rri.ItemID] = rri.ID
	}
	for idx, crt := range crts {
		ip := rriID2Resource[rriItemId2RriID[crt.RRIItemID]].IP
		switch strings.ToLower(crt.Component) {
		case "tidb":
			topo.TiDBServers[ip] = crts[idx]
		case "tikv":
			topo.TiKVServers[ip] = crts[idx]
		case "pd":
			topo.PDServers[ip] = crts[idx]
		case "monitoring":
			topo.MonitoringServers[ip] = crts[idx]
		case "grafana":
			topo.GrafanaServers[ip] = crts[idx]
		default:
			return fmt.Errorf("unknown component type %s", crt.Component)
		}
	}

	yaml := buildTopologyYaml(topo)
	if err := deployCluster(yaml, name, cr.Version); err != nil {
		return errors.Trace(err)
	}
	if err := startCluster(name); err != nil {
		return errors.Trace(err)
	}
	return nil
}

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

func TryDestroyCluster(name string) error {
	output, err := util.Command("", "tiup", "cluster", "destroy", name, "-y")
	if err != nil {
		return fmt.Errorf("destroy cluster failed, err: %v, output: %s", err, output)
	}
	return nil
}

func deployCluster(yaml, name, version string) error {
	file, err := ioutil.TempFile("", "topo")
	if err != nil {
		return errors.Trace(err)
	}
	defer file.Close()
	file.WriteString(yaml)

	output, err := util.Command("", "tiup", "cluster", "deploy", "-y", name, version, file.Name())
	if err != nil {
		return fmt.Errorf("deploy cluster failed, err: %v, output: %s", err, output)
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
	for host, config := range t.MonitoringServers {
		topo.WriteString(fmt.Sprintf(`
  - host: %s
    deploy_dir: "%s/data/prometheus-8249"`, host, config.DeployPath))
	}

	topo.WriteString(`

grafana_servers:`)
	for host, config := range t.GrafanaServers {
		topo.WriteString(fmt.Sprintf(`
  - host: %s
    deploy_dir: "%s/data/grafana-3000"`, host, config.DeployPath))
	}
	return topo.String()
}
