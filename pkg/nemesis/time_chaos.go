package nemesis

import (
	"fmt"
	"math/rand"
)

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
	skewTimeMap = make([]uint, 0)
	skewTimeMap = append(skewTimeMap, 0)
	skewTimeMap = append(skewTimeMap, 100)
	skewTimeMap = append(skewTimeMap, 200)
	skewTimeMap = append(skewTimeMap, 250)
	skewTimeMap = append(skewTimeMap, 500)
	skewTimeMap = append(skewTimeMap, 5000)

	skewTimeStrMap = map[string]TimeChaosLevels{
		"small-skews": SmallSkews,
		"subcritical-skews": SubCriticalSkews,
		"critical-skews": CriticalSkews,
		"big-skews": BigSkews,
		"huge-skews": HugeSkews,
		"strobe-skews": StrobeSkews,
	}
}

// Panic: if chaos not in skewTimeStrMap, then panic.
func timeChaosLevel(chaos string) TimeChaosLevels {
	var level TimeChaosLevels
	var ok bool
	if level, ok = skewTimeStrMap[chaos]; ok {
		panic(fmt.Sprintf("timeChaosLevel receive chaos %s, which is not supported.", chaos))
	}
	return level
}

func selectChaosDuration(levels TimeChaosLevels, durationType ChaosDurationType) (int, int) {
	var secs, nanoSec int
	if levels == StrobeSkews {
		deltaMs := rand.Intn(200)
		nanoSec = deltaMs * int(MsToNS)
	} else {
		var lastVal uint
		if durationType == FromLast {
			lastVal = skewTimeMap[levels]
		} else {
			lastVal = 0
		}

		deltaMs := uint(rand.Int31n(int32(skewTimeMap[levels+1]-lastVal)*2)) + 2*lastVal - skewTimeMap[levels+1]
		if deltaMs > SecToNS {
			secs = int(deltaMs) / int(SecToNS)
			deltaMs = deltaMs % SecToNS
		}
		nanoSec = int(deltaMs * MsToNS)
	}

	return secs, nanoSec
}

func selectDefaultChaosDuration(levels TimeChaosLevels) (int, int) {
	return selectChaosDuration(levels, FromLast)
}

type timeChaosGenerator struct {
}
