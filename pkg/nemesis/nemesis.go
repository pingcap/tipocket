package nemesis

import (
	"math/rand"

	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
)

type k8sNemesisClient struct {
	cli *Chaos
}

func init() {
	client := k8sNemesisClient{New(tests.TestClient.Cli)}
	core.RegisterNemesis(kill{client})
	core.RegisterNemesis(podKill{client})
	core.RegisterNemesis(containerKill{client})
	core.RegisterNemesis(networkPartition{client})
	core.RegisterNemesis(netem{client})
	core.RegisterNemesis(scaling{client})
	core.RegisterNemesis(timeChaos{client})
	core.RegisterNemesis(scheduler{})
	core.RegisterNemesis(newLeaderShuffler())
	core.RegisterNemesis(iochaos{client})
	core.RegisterNemesis(dynamicConfig{})
}

func shuffleIndices(n int) []int {
	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = i
	}
	for i := len(indices) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		indices[i], indices[j] = indices[j], indices[i]
	}

	return indices
}
