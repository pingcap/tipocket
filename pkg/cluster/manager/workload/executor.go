package workload

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
	"github.com/pingcap/tipocket/pkg/cluster/manager/util"
)

func TryRunWorkload(name string,
	resources []types.Resource,
	rris []*types.ResourceRequestItem,
	wr *types.WorkloadRequest,
	envs map[string]string,
) ([]byte, []byte, error) {

	rriID2Resource := make(map[uint]types.Resource)
	rriItemId2RriID := make(map[uint]uint)
	component2Resources := make(map[string][]types.Resource)
	// resource request item id -> resource
	for _, re := range resources {
		rriID2Resource[re.RRIID] = re
	}
	// resource request item item_id ->  resource request item id
	for _, rri := range rris {
		rriItemId2RriID[rri.ItemID] = rri.ID
		for _, component := range strings.Split(rri.Components, "|") {
			if _, ok := component2Resources[component]; !ok {
				component2Resources[component] = make([]types.Resource, 0)
			}
			component2Resources[component] = append(component2Resources[component], rriID2Resource[rri.ID])
		}
	}
	resource := rriID2Resource[rriItemId2RriID[wr.RRIItemID]]
	host := resource.IP

	if envs == nil {
		envs = make(map[string]string)
	}

	envs["CLUSTER_NAME"] = name
	envs["API_SERVER"] = fmt.Sprintf("http://%s", util.Addr)
	if rs, err := randomResource(component2Resources["pd"]); err != nil {
		return nil, nil, errors.Trace(err)
	} else {
		envs["PD_ADDR"] = fmt.Sprintf("%s:2379", rs.IP)
	}
	if rs, err := randomResource(component2Resources["tidb"]); err != nil {
		return nil, nil, errors.Trace(err)
	} else {
		envs["TIDB_ADDR"] = fmt.Sprintf("%s:4000", rs.IP)
	}
	if rs, err := randomResource(component2Resources["monitoring"]); err != nil {
		return nil, nil, errors.Trace(err)
	} else {
		envs["PROM_ADDR"] = fmt.Sprintf("http://%s:9090", rs.IP)
	}

	dockerExecutor, err := util.NewDockerExecutor(fmt.Sprintf("tcp://%s:2375", host))
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	return dockerExecutor.Run(wr.DockerImage, envs, wr.Cmd, wr.Args...)
}

func randomResource(rs []types.Resource) (types.Resource, error) {
	if len(rs) == 0 {
		return types.Resource{}, fmt.Errorf("expect non-empty resources")
	}
	return rs[rand.Intn(len(rs))], nil
}
