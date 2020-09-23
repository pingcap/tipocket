package workload

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/artifacts"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
	"github.com/pingcap/tipocket/pkg/cluster/manager/util"
)

// RunWorkload creates the workload docker container and injects necessary
// environment variables
func RunWorkload(
	cr *types.ClusterRequest,
	resources []*types.Resource,
	rris []*types.ResourceRequestItem,
	wr *types.WorkloadRequest,
	artifactUUID string,
	envs map[string]string,
) (dockerExecutor *util.DockerExecutor, containerID string, out *bytes.Buffer, err error) {
	id2Resource := make(map[uint]*types.Resource)
	rriItemID2RID := make(map[uint]uint)
	component2Resources := make(map[string][]*types.Resource)
	// resource request item id -> resource
	for idx, re := range resources {
		id2Resource[re.ID] = resources[idx]
	}
	// resource request item item_id -> resource request item id
	for _, rri := range rris {
		rriItemID2RID[rri.ItemID] = rri.RID
		for _, component := range strings.Split(rri.Components, "|") {
			if _, ok := component2Resources[component]; !ok {
				component2Resources[component] = make([]*types.Resource, 0)
			}
			component2Resources[component] = append(component2Resources[component], id2Resource[rri.RID])
		}
	}
	resource := id2Resource[rriItemID2RID[wr.RRIItemID]]
	host := resource.IP
	if envs == nil {
		envs = make(map[string]string)
	}
	var (
		rs *types.Resource
	)
	envs["CLUSTER_ID"] = fmt.Sprintf("%d", cr.ID)
	envs["CLUSTER_NAME"] = cr.Name
	envs["API_SERVER"] = fmt.Sprintf("http://%s", util.Addr)
	envs["ARTIFACT_URL"] = fmt.Sprintf("%s/%s/%s", util.S3Endpoint, artifacts.ArtifactDownloadPath(cr.ID), artifactUUID)

	if rs, err = randomResource(component2Resources["pd"]); err != nil {
		return nil, "", nil, errors.Trace(err)
	}
	envs["PD_ADDR"] = fmt.Sprintf("%s:2379", rs.IP)

	if len(component2Resources["tidb"]) != 0 {
		if rs, err = randomResource(component2Resources["tidb"]); err != nil {
			return nil, "", nil, errors.Trace(err)
		}
		envs["TIDB_ADDR"] = fmt.Sprintf("%s:4000", rs.IP)
	}

	if rs, err = randomResource(component2Resources["prometheus"]); err != nil {
		return nil, "", nil, errors.Trace(err)
	}
	envs["PROM_ADDR"] = fmt.Sprintf("http://%s:9090", rs.IP)

	dockerExecutor, err = util.NewDockerExecutor(fmt.Sprintf("tcp://%s:2375", host))
	if err != nil {
		return nil, "", nil, errors.Trace(err)
	}
	if o, err := RestoreDataIfConfig(wr, envs, dockerExecutor); err != nil {
		return nil, "", o, errors.Trace(err)
	}
	containerID, out, err = dockerExecutor.Run(wr.DockerImage, envs, wr.Cmd, wr.Args...)
	return
}

// RestoreDataIfConfig ...
func RestoreDataIfConfig(wr *types.WorkloadRequest, envs map[string]string, dockerExecutor *util.DockerExecutor) (*bytes.Buffer, error) {
	if wr.RestorePath != nil {
		envs["S3_ENDPOINT"] = fmt.Sprintf("http://%s", util.S3Endpoint)
		envs["AWS_ACCESS_KEY_ID"] = util.AwsAccessKeyID
		envs["AWS_SECRET_ACCESS_KEY"] = util.AwsSecretAccessKey

		// FIXME(mahjonp): replace with pingcap/br in future
		_, o, err := dockerExecutor.Run("mahjonp/br",
			envs,
			&[]string{"/bin/bash"}[0],
			"-c", fmt.Sprintf("bin/br restore full --pd $PD_ADDR --storage s3://"+*wr.RestorePath+" --s3.endpoint $S3_ENDPOINT --send-credentials-to-tikv=true"))
		if err != nil {
			return o, errors.Trace(err)
		}
		return o, nil
	}
	return nil, nil
}

func randomResource(rs []*types.Resource) (*types.Resource, error) {
	if len(rs) == 0 {
		return nil, fmt.Errorf("expect non-empty resources")
	}
	return rs[rand.Intn(len(rs))], nil
}
