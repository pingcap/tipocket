package nemesis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
)

// Note: TimeChaosLevels means the level of defined time chaos like jepsen.
// 	It's an integer start from 0.
type TimeChaosLevels = int

const (
	SmallSkews TimeChaosLevels = iota
	SubCriticalSkews
	CriticalSkews
	BigSkews
	HugeSkews
	// Note: StrobeSkews currently should be at end of iota system,
	//  because TimeChaosLevels will be used as slice index.
	StrobeSkews
)

const StrobeSkewsBios = 200

type ChaosDurationType int

const (
	FromZero ChaosDurationType = iota
	FromLast
)

const MsToNS uint = 1000000
const SecToNS uint = 1e9
const SecToMS uint = SecToNS / MsToNS

// skewTimeMap stores the
var skewTimeMap []uint
var skewTimeStrMap map[string]TimeChaosLevels

func init() {
	skewTimeMap = []uint{
		0,
		100,
		200,
		250,
		500,
		5000,
	}

	skewTimeStrMap = map[string]TimeChaosLevels{
		"small_skews":       SmallSkews,
		"subcritical_skews": SubCriticalSkews,
		"critical_skews":    CriticalSkews,
		"big_skews":         BigSkews,
		"huge_skews":        HugeSkews,
		"strobe-skews":      StrobeSkews,
	}
}

// Panic: if chaos not in skewTimeStrMap, then panic.
func timeChaosLevel(chaos string) TimeChaosLevels {
	var level TimeChaosLevels
	var ok bool
	if level, ok = skewTimeStrMap[chaos]; !ok {
		panic(fmt.Sprintf("timeChaosLevel receive chaos %s, which is not supported.", chaos))
	}
	return level
}

// selectChaosDuration selects a random (seconds, nano seconds) form Level and duration type.
// `TimeChaosLevels` is ported from Jepsen, which means different time bios.
// `ChaosDurationType` means start from zero ([0, 200ms]) or start from last level [100ms, 200ms].
func selectChaosDuration(levels TimeChaosLevels, durationType ChaosDurationType) (int, int) {
	var secs, nanoSec int
	if levels == StrobeSkews {
		deltaMs := rand.Intn(StrobeSkewsBios)
		nanoSec = deltaMs * int(MsToNS)
	} else {
		var lastVal uint
		if durationType == FromLast {
			lastVal = skewTimeMap[levels]
		} else {
			lastVal = 0
		}

		// [-skewTimeMap[levels+1], -lastVal] Union [lastVal, skewTimeMap[levels+1]]
		deltaMs := uint(rand.Intn(int(skewTimeMap[levels+1]-lastVal))) + lastVal

		if uint(math.Abs(float64(deltaMs))) > SecToNS {
			secs = int(deltaMs) / int(SecToNS)
			deltaMs = deltaMs % SecToNS
		}
		nanoSec = int(deltaMs * MsToNS)

		if rand.Int()%2 == 1 {
			nanoSec = -nanoSec
			secs = -secs
		}
	}

	return secs, nanoSec
}

func selectDefaultChaosDuration(levels TimeChaosLevels) (int, int) {
	return selectChaosDuration(levels, FromLast)
}

type timeChaosGenerator struct {
	name string
}

func (t timeChaosGenerator) Generate(nodes []types.Node) []*core.NemesisOperation {
	ops := make([]*core.NemesisOperation, len(nodes))

	for idx := range nodes {
		node := nodes[idx]
		secs, nanoSecs := selectChaosDuration(timeChaosLevel(t.name), FromLast)
		ops = append(ops, &core.NemesisOperation{
			Type:        core.TimeChaos,
			Node:        &node,
			InvokeArgs:  []interface{}{secs, nanoSecs},
			RecoverArgs: []interface{}{secs, nanoSecs},
			RunTime:     time.Second * time.Duration(rand.Intn(120)+60),
		})
	}

	return ops
}

func (t timeChaosGenerator) Name() string {
	return t.name
}

// NewTimeChaos generate a time chaos.
func NewTimeChaos(name string) core.NemesisGenerator {
	return timeChaosGenerator{name: name}
}

type timeChaos struct {
	k8sNemesisClient
}

func (t timeChaos) Invoke(ctx context.Context, node *types.Node, args ...interface{}) error {
	if len(args) != 2 {
		panic("args number error")
	}
	secs, ok := args[0].(int)
	if !ok {
		return errors.New("the first argument of timeChaos.Invoke should be an integer")
	}
	nanoSecs, ok := args[1].(int)
	if !ok {
		return errors.New("the second argument of timeChaos.Invoke should be an integer")
	}
	log.Printf("Creating time-chaos with node %s(ns:%s)\n", node.PodName, node.Namespace)
	timeChaos := buildTimeChaos(node.Namespace, node.Namespace, node.PodName, int64(secs), int64(nanoSecs))
	return t.cli.ApplyTimeChaos(ctx, &timeChaos)
}

func (t timeChaos) Recover(ctx context.Context, node *types.Node, args ...interface{}) error {
	if len(args) != 2 {
		panic("args number error")
	}
	secs, ok := args[0].(int)
	if !ok {
		return errors.New("the first argument of timeChaos.Invoke should be an integer")
	}
	nanoSecs, ok := args[1].(int)
	if !ok {
		return errors.New("the second argument of timeChaos.Invoke should be an integer")
	}
	log.Printf("Recovering time-chaos with node %s(ns:%s)\n", node.PodName, node.Namespace)
	timeChaos := buildTimeChaos(node.Namespace, node.Namespace, node.PodName, int64(secs), int64(nanoSecs))
	return t.cli.CancelTimeChaos(ctx, &timeChaos)
}

func (t timeChaos) Name() string {
	return string(core.TimeChaos)
}

func buildTimeChaos(ns, chaosNs, podName string, secs, nanoSecs int64) chaosv1alpha1.TimeChaos {
	pods := make(map[string][]string)
	pods[ns] = []string{podName}
	return chaosv1alpha1.TimeChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.Join([]string{ns, podName, "time-chaos"}, "-"),
			Namespace: chaosNs,
		},
		// Note: currently only one mode support,
		//  but we can add another modes in the future.
		Spec: chaosv1alpha1.TimeChaosSpec{
			Mode: chaosv1alpha1.OnePodMode,
			Selector: chaosv1alpha1.SelectorSpec{
				Pods: pods,
			},
			TimeOffset: chaosv1alpha1.TimeOffset{
				Sec:  secs,
				NSec: nanoSecs,
			},
		},
	}
}
