package deploy

import (
	"bytes"
	"fmt"
	"github.com/juju/errors"
	"io/ioutil"
	"strings"

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

func TryDeploy(resources []types.Resource, cr *types.ClusterRequest, crts []types.ClusterRequestTopology) error {
	topo := &Topology{
		Config:            cr.Config,
		PDServers:         make(map[string]*types.ClusterRequestTopology),
		TiKVServers:       make(map[string]*types.ClusterRequestTopology),
		TiDBServers:       make(map[string]*types.ClusterRequestTopology),
		MonitoringServers: make(map[string]*types.ClusterRequestTopology),
		GrafanaServers:    make(map[string]*types.ClusterRequestTopology),
	}
	rriID2Resource := make(map[uint]types.Resource)
	for _, re := range resources {
		rriID2Resource[re.RRIID] = re
	}
	for idx, crt := range crts {
		switch strings.ToLower(crt.Component) {
		case "tidb":
			topo.TiDBServers[rriID2Resource[crt.RRIItemID].IP] = &crts[idx]
		case "tikv":
			topo.TiKVServers[rriID2Resource[crt.RRIItemID].IP] = &crts[idx]
		case "pd":
			topo.PDServers[rriID2Resource[crt.RRIItemID].IP] = &crts[idx]
		case "monitoring":
			topo.MonitoringServers[rriID2Resource[crt.RRIItemID].IP] = &crts[idx]
		case "grafana":
			topo.GrafanaServers[rriID2Resource[crt.RRIItemID].IP] = &crts[idx]
		default:
			return fmt.Errorf("unknown component type %s", crt.Component)
		}
	}

	yaml := buildTopologyYaml(topo)

	file, err := ioutil.TempFile("", "topo")
	if err != nil {
		return errors.Trace(err)
	}
	defer file.Close()
	file.WriteString(yaml)
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
    deploy_dir: "%s/deploy/tidb-4000"
    data_dir: "%s/data/tidb-4000"`, host, config.DeployPath, config.DeployPath))
	}

	topo.WriteString(`

tikv_servers:`)
	for host, config := range t.TiDBServers {
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
