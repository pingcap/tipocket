package txn

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

// DistMode contains uniform or exponential distribution
type DistMode = int

const (
	// Uniform means uniform distribution, every key has an equal probability of appearing.
	Uniform DistMode = iota
	// Exponential means that key i in the current key pool chosen.
	Exponential
)

// KeyDist means the distribution of the keys. Defaults to :exponential.
type KeyDist struct {
	DistMode DistMode
}

// WrTxnOpts ...
type WrTxnOpts struct {
	KeyDist KeyDist
	// The base for an exponential distribution.
	// Defaults to 2, so the first key is twice as likely as the second,
	//  which is twice as likely as the third, etc.
	KeyDistBase     uint
	KeyCount        uint
	MinTxnLength    uint
	MaxTxnLength    uint
	MaxWritesPerKey uint
}

// DefaultWrTxnOpts returns default opts
func DefaultWrTxnOpts() WrTxnOpts {
	return WrTxnOpts{
		KeyDist: KeyDist{
			DistMode: Uniform,
		},
		KeyCount:        5,
		MinTxnLength:    1,
		MaxTxnLength:    4,
		MaxWritesPerKey: 16,
	}
}

func powInt(base, exp uint) uint {
	var result uint
	result = 1
	for {
		if exp != 0 {
			result *= base
		}
		exp /= 2
		if exp == 0 {
			break
		}
		base *= base
	}

	return result
}

func (opts WrTxnOpts) keyDistScale() uint {
	return ((powInt(opts.KeyDistBase, opts.KeyCount) - 1) * opts.KeyDistBase) / (opts.KeyDistBase - 1)
}

// MopIterator records the state of the mop generator
type MopIterator struct {
	opts       WrTxnOpts
	activeKeys WrTxnState
}

// Next iterates next element
//  If the operation is `read`, it will just return [:r key nil]
//  If the operation is `write`, it will just return [:w key value]
func (mIter *MopIterator) Next() []core.Mop {
	//keyDistScale := mIter.opts.keyDistScale()
	length := uint(rand.Int63nRange(int64(int(mIter.opts.MinTxnLength)), int64(mIter.opts.MaxTxnLength)) + 1)

	var mops []core.Mop
	for uint(len(mops)) < length {
		// It's supporting for Exponential
		//ki := uint64(math.Floor(math.Log(float64(uint(rand.Intn(int(keyDistScale)))+mIter.opts.KeyDistBase)) - 1))
		var writeVersion int
		var k string
		switch mIter.opts.KeyDist.DistMode {
		case Exponential:
			panic("not implemented")
		case Uniform:
			k, writeVersion = mIter.activeKeys.randomGet()
		}
		if uint(writeVersion) > mIter.opts.MaxWritesPerKey {
			// we activate a new key and replace old key with it
			delete(mIter.activeKeys.keyRecord, k)
			k = mustUtoa(mIter.activeKeys.maxKey)
			mIter.activeKeys.maxKey++

			mIter.activeKeys.keyRecord[k] = 1
			writeVersion = 1
		}
		var mop core.Mop
		switch rand.Intn(2) {
		case 0:
			mop = core.Append(
				k,
				writeVersion,
			)
			mIter.activeKeys.keyRecord[k] = mIter.activeKeys.keyRecord[k] + 1
		case 1:
			mop = core.Read(k, nil)
		}
		mops = append(mops, mop)
	}
	return mops
}

// WrTxnState ...
type WrTxnState struct {
	// map for [Key, WriteCnt]
	keyRecord map[string]int
	maxKey    uint
}

func (s WrTxnState) randomGet() (string, int) {
	i := rand.Intn(len(s.keyRecord))
	for k, v := range s.keyRecord {
		if i == 0 {
			return k, v
		}
		i--
	}
	panic("error")
}

// WrTxn ...
func WrTxn(opts WrTxnOpts) *MopIterator {
	state := WrTxnState{keyRecord: map[string]int{}, maxKey: opts.KeyCount}
	for i := 0; uint(i) < opts.KeyCount; i++ {
		state.keyRecord[mustItoa(i)] = 1
	}
	return &MopIterator{opts: opts, activeKeys: state}
}

// WrTxnWithDefaultOpts ...
func WrTxnWithDefaultOpts() *MopIterator {
	return WrTxn(DefaultWrTxnOpts())
}

func mustItoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func mustUtoa(u uint) string {
	return fmt.Sprintf("%d", u)
}
