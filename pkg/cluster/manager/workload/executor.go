package workload

import (
	"fmt"

	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
	"github.com/pingcap/tipocket/pkg/cluster/manager/util"
)

func TryRunWorkload(name string,
	resources []types.Resource,
	rris []*types.ResourceRequestItem,
	wr *types.WorkloadRequest,
) ([]byte, []byte, error) {

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
	resource := rriID2Resource[rriItemId2RriID[wr.RRIItemID]]
	host := resource.IP
	cmd := fmt.Sprintf("docker run %s %s %s", wr.DockerImage, wr.Cmd, wr.ARGS)
	sshExecutor := util.NewSSHExecutor(util.SSHConfig{
		Host:       host,
		Port:       22,
		User:       resource.Username,
		KeyFile:    util.SSHKeyPath(),
	})
	return sshExecutor.Execute(cmd)
}
