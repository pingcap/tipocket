package nemesis

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

type netemChaosGenerator struct {
	name string
}

type loss struct {

}

func (l loss) name() string {
	panic("implement me")
}

func (l loss) template() chaosv1alpha1.NetworkChaosSpec {
	panic("implement me")
}

func (l loss) defaultTemplate() chaosv1alpha1.NetworkChaosSpec {
	panic("implement me")
}

type delay struct {

}


type netemChaos interface {
	// Name returns chaos name like "loss" "delay"
	name() string
	template() chaosv1alpha1.NetworkChaosSpec
	defaultTemplate() chaosv1alpha1.NetworkChaosSpec
}

func selectNetem(name string) netemChaos {
	switch name {
	case "loss":
		return loss{}
	case "delay":
		panic("impl me")
	case "duplicate":
		panic("impl me")
	case "corrupt":
		panic("impl me")
	}
}

func (g netemChaosGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	n := 1
	switch g.name {

	}
	panic("not implemented")
}


func (g netemChaosGenerator) Name() string {
	return g.name
}


type netem struct {
	k8sNemesisClient
	netemType chaosv1alpha1.NetworkChaosAction
}

func (n netem) Name() string {
	return string(n.netemType)
}
