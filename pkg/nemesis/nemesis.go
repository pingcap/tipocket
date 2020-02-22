package nemesis

import (
	"math/rand"
	"os"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"
)

type k8sNemesisClient struct {
	cli *Chaos
}

func init() {
	client := k8sNemesisClient{mustCreateClient()}
	core.RegisterNemesis(kill{client})
	core.RegisterNemesis(podKill{client})
	core.RegisterNemesis(containerKill{client})
	core.RegisterNemesis(networkPartition{client})
}

func createClient() (*Chaos, error) {
	conf, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}
	kubeCli, err := fixture.BuildGenericKubeClient(conf)
	if err != nil {
		return nil, err
	}
	return New(kubeCli), nil
}

func mustCreateClient() *Chaos {
	cli, err := createClient()
	if err != nil {
		panic(err)
	}
	return cli
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
