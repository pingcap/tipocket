package nemesis

import (
	"math/rand"
	"time"

	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
)

var (
	r           = rand.New(rand.NewSource(time.Now().UnixNano()))
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
)

func randK8sObjectName() string {
	b := make([]rune, 7)
	for i := range b {
		b[i] = letterRunes[r.Intn(len(letterRunes))]
	}
	return string(b)
}

type k8sNemesisClient struct {
	cli *Chaos
}

func init() {
	// most kinds of nemesis depends on chaos-mesh or tidb-operator
	if tests.TestClient.Cli != nil {
		client := k8sNemesisClient{New(tests.TestClient.Cli)}
		core.RegisterNemesis(kill{client})
		core.RegisterNemesis(podKill{client})
		core.RegisterNemesis(containerKill{client})
		core.RegisterNemesis(networkPartition{client})
		core.RegisterNemesis(netem{client})
		core.RegisterNemesis(timeChaos{client})
		core.RegisterNemesis(iochaos{client})
		core.RegisterNemesis(scaling{client})
	}
	core.RegisterNemesis(scheduler{})
	core.RegisterNemesis(newLeaderShuffler())
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
