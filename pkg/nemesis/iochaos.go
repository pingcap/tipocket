package nemesis

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
)

const (
	chaosfs = "chaosfs-"

	// empty string means random errno in errno or mixed io chaos
	randomErrno = ""
)

type IOChaosGenerator struct {
	name string
}

// NewIOChaosGenerator create an io chaos.
func NewIOChaosGenerator(name string) core.NemesisGenerator {
	return IOChaosGenerator{name: name}
}

// Generate generates nemesises based on nemesis type.
func (g IOChaosGenerator) Generate(nodes []clusterTypes.Node) []*core.NemesisOperation {
	duration := time.Second * time.Duration(rand.Intn(120)+60)

	// since the nemesis name is in the form of
	// like delay_tikv or errno_pd, we can split
	// the nemesis name to get the io chaos type
	// and component
	parts := strings.Split(g.name, "_")
	chaos := selectIOChaos(parts[0])

	filteredNodes := filterComponent(nodes, clusterTypes.Component(parts[1]))

	return []*core.NemesisOperation{{
		Type:        core.IOChaos,
		InvokeArgs:  []interface{}{chaos, filteredNodes},
		RecoverArgs: []interface{}{chaos, filteredNodes},
		RunTime:     duration,
	}}
}

func (g IOChaosGenerator) Name() string {
	return g.name
}

type ioDelay struct{}

func (d ioDelay) ioChaosType() chaosv1alpha1.IOChaosAction {
	return chaosv1alpha1.IODelayAction
}

// The first arg means loss, the second args means correlation.
func (d ioDelay) template(
	ns string,
	pods []string,
	podMode chaosv1alpha1.PodMode,
	layer chaosv1alpha1.IOLayer,
	config string,
	methods []string,
	args ...string,
) chaosv1alpha1.IoChaosSpec {

	if len(args) != 2 {
		panic("args number error")
	}
	return chaosv1alpha1.IoChaosSpec{
		Action: chaosv1alpha1.IODelayAction,
		Selector: chaosv1alpha1.SelectorSpec{
			Namespaces: []string{ns},
			Pods:       map[string][]string{ns: pods},
		},
		Layer:      layer,
		Mode:       podMode,
		ConfigName: config,
		Delay:      args[0],
		Percent:    args[1],
		Methods:    methods,
	}
}

func (d ioDelay) defaultTemplate(ns, configMap string, pods []string) chaosv1alpha1.IoChaosSpec {
	return d.template(ns, pods, chaosv1alpha1.OnePodMode,
		chaosv1alpha1.FileSystemLayer, configMap, nil, "50ms", "50")
}

type ioErrno struct{}

func (e ioErrno) ioChaosType() chaosv1alpha1.IOChaosAction {
	return chaosv1alpha1.IOErrnoAction
}

func (e ioErrno) template(
	ns string,
	pods []string,
	podMode chaosv1alpha1.PodMode,
	layer chaosv1alpha1.IOLayer,
	config string,
	methods []string,
	args ...string,
) chaosv1alpha1.IoChaosSpec {

	if len(args) != 2 {
		panic("args number error")
	}
	return chaosv1alpha1.IoChaosSpec{
		Action: chaosv1alpha1.IOErrnoAction,
		Selector: chaosv1alpha1.SelectorSpec{
			Namespaces: []string{ns},
			Pods:       map[string][]string{ns: pods},
		},
		Layer:      layer,
		Mode:       podMode,
		ConfigName: config,
		Errno:      args[0],
		Percent:    args[1],
		Methods:    methods,
	}
}

func (e ioErrno) defaultTemplate(ns, configMap string, pods []string) chaosv1alpha1.IoChaosSpec {
	return e.template(ns, pods, chaosv1alpha1.OnePodMode,
		chaosv1alpha1.FileSystemLayer, configMap, nil, randomErrno, "50")
}

type ioReadEerr struct{}

func (e ioReadEerr) ioChaosType() chaosv1alpha1.IOChaosAction {
	return chaosv1alpha1.IOErrnoAction
}

func (e ioReadEerr) template(
	ns string,
	pods []string,
	podMode chaosv1alpha1.PodMode,
	layer chaosv1alpha1.IOLayer,
	config string,
	methods []string,
	args ...string,
) chaosv1alpha1.IoChaosSpec {

	if len(args) != 2 {
		panic("args number error")
	}
	return chaosv1alpha1.IoChaosSpec{
		Action: chaosv1alpha1.IOErrnoAction,
		Selector: chaosv1alpha1.SelectorSpec{
			Namespaces: []string{ns},
			Pods:       map[string][]string{ns: pods},
		},
		Layer:      layer,
		Mode:       podMode,
		ConfigName: config,
		Errno:      args[0],
		Percent:    args[1],
		Methods:    methods,
	}
}

func (e ioReadEerr) defaultTemplate(ns, configMap string, pods []string) chaosv1alpha1.IoChaosSpec {
	return e.template(ns, pods, chaosv1alpha1.OnePodMode,
		chaosv1alpha1.FileSystemLayer, configMap, []string{"read"}, randomErrno, "50")
}

type ioMixed struct{}

func (m ioMixed) ioChaosType() chaosv1alpha1.IOChaosAction {
	return chaosv1alpha1.IOMixedAction
}

func (m ioMixed) template(
	ns string,
	pods []string,
	podMode chaosv1alpha1.PodMode,
	layer chaosv1alpha1.IOLayer,
	configName string,
	methods []string,
	args ...string,
) chaosv1alpha1.IoChaosSpec {

	if len(args) != 3 {
		panic("args number error")
	}
	return chaosv1alpha1.IoChaosSpec{
		Action: chaosv1alpha1.IOMixedAction,
		Selector: chaosv1alpha1.SelectorSpec{
			Namespaces: []string{ns},
			Pods:       map[string][]string{ns: pods},
		},
		Layer:      layer,
		Mode:       podMode,
		ConfigName: configName,
		Delay:      args[0],
		Errno:      args[1],
		Percent:    args[2],
		Methods:    methods,
	}
}

func (m ioMixed) defaultTemplate(ns, configName string, pods []string) chaosv1alpha1.IoChaosSpec {
	return m.template(ns, pods, chaosv1alpha1.OnePodMode,
		chaosv1alpha1.FileSystemLayer, configName, nil, "50ms", randomErrno, "50")
}

type ioChaos interface {
	ioChaosType() chaosv1alpha1.IOChaosAction
	template(ns string, pods []string, podMode chaosv1alpha1.PodMode,
		layer chaosv1alpha1.IOLayer, configMap string, methods []string, args ...string) chaosv1alpha1.IoChaosSpec
	defaultTemplate(ns, configMap string, pods []string) chaosv1alpha1.IoChaosSpec
}

func selectIOChaos(name string) ioChaos {
	switch name {
	case "delay":
		return ioDelay{}
	case "errno":
		return ioErrno{}
	case "mixed":
		return ioMixed{}
	case "readerr":
		return ioReadEerr{}
	default:
		panic("unsupported io chaos action")
	}
}

type iochaos struct {
	k8sNemesisClient
}

func (n iochaos) extractChaos(args ...interface{}) chaosv1alpha1.IoChaos {
	if len(args) != 2 {
		panic("ioChaos arg number is wrong")
	}
	var c ioChaos
	var ok bool
	var nodes []clusterTypes.Node

	if c, ok = args[0].(ioChaos); !ok {
		panic("ioChaos get wrong type")
	}

	if nodes, ok = args[1].([]clusterTypes.Node); !ok {
		panic("nodes get wrong type")
	}

	node := nodes[0]
	componentType := string(node.Component)

	podNames := make([]string, len(nodes))
	for i := range nodes {
		podNames[i] = nodes[i].PodName
	}

	ioChaosSpec := c.defaultTemplate(node.Namespace, chaosfs+componentType, podNames)
	return chaosv1alpha1.IoChaos{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", string(c.ioChaosType()), componentType),
			Namespace: node.Namespace,
		},
		Spec: ioChaosSpec,
	}
}

func (n iochaos) Invoke(_ context.Context, _ *clusterTypes.Node, args ...interface{}) error {
	chaosSpec := n.extractChaos(args...)
	log.Printf("Invoke io chaos %s at ns:%s\n", chaosSpec.Name, chaosSpec.Namespace)
	return n.cli.ApplyIOChaos(&chaosSpec)
}

func (n iochaos) Recover(_ context.Context, _ *clusterTypes.Node, args ...interface{}) error {
	chaosSpec := n.extractChaos(args...)
	log.Printf("Recover io chaos %s at ns:%s\n", chaosSpec.Name, chaosSpec.Namespace)
	return n.cli.CancelIOChaos(&chaosSpec)
}

func (n iochaos) Name() string {
	return string(core.IOChaos)
}
