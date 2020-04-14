package txn

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

type DistMode = int

const (
	// Optional string field for Uniform. means every key has an equal probability of appearing.
	Uniform DistMode = iota
	// Exponential means that key i in the current key pool chosen.
	Exponential
)

// KeyDist means the distribution of the keys. Defaults to :exponential.
type KeyDist struct {
	DistMode DistMode
}

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

func DefaultWrTxnOpts() WrTxnOpts {
	return WrTxnOpts{
		KeyDist: KeyDist{
			DistMode: Uniform,
		},
		KeyCount:        3,
		MinTxnLength:    1,
		MaxTxnLength:    2,
		MaxWritesPerKey: 32,
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

type MopIterator struct {
	opts       WrTxnOpts
	activeKeys WrTxnState
}

// Note: It will return array of mop
//  If the operation is `read`, it will just return [:r key nil]
//  If the operation is `write`, it will just return [:w key value]
func (mIter *MopIterator) Next() []core.Mop {
	//keyDistScale := mIter.opts.keyDistScale()
	length := uint(rand.Int63nRange(int64(int(mIter.opts.MinTxnLength)), int64(mIter.opts.MaxTxnLength)) + 1)

	var mops []core.Mop
	for uint(len(mops)) < length {
		// It's supporting for Exponential
		//ki := uint64(math.Floor(math.Log(float64(uint(rand.Intn(int(keyDistScale)))+mIter.opts.KeyDistBase)) - 1))
		var k, writeVersion uint
		switch mIter.opts.KeyDist.DistMode {
		case Exponential:
			panic("not implemented")
		case Uniform:
			k, writeVersion = mIter.activeKeys.randomGet()
		}
		// TODO: here may cause a bug, so please pay attention and change the logic here.
		if writeVersion >= mIter.opts.MaxWritesPerKey {
			continue
		}
		var mop core.Mop
		switch rand.Intn(2) {
		case 0:
			mop = core.Append{
				Key:   fmt.Sprintf("%d", k),
				Value: core.MopValueType(writeVersion),
			}
			mIter.activeKeys.keyRecord[k] = mIter.activeKeys.keyRecord[k] + 1
		case 1:
			mop = core.Read{
				Key: fmt.Sprintf("%d", k),
			}
		}
		mops = append(mops, mop)
	}
	return mops
}

type WrTxnState struct {
	// map for [TxnID, WriteCnt]
	keyRecord map[uint]uint
}

func (s WrTxnState) randomGet() (uint, uint) {
	i := rand.Intn(len(s.keyRecord))
	for k, v := range s.keyRecord {
		if i == 0 {
			return k, v
		}
		i--
	}
	panic("error")
}

func WrTxn(opts WrTxnOpts, state WrTxnState) *MopIterator {
	return &MopIterator{opts: opts, activeKeys: state}
}

func WrTxnWithDefaultOpts(state WrTxnState) *MopIterator {
	return WrTxn(DefaultWrTxnOpts(), state)
}

func WrTxnWithoutState(opts WrTxnOpts) *MopIterator {
	state := WrTxnState{keyRecord: map[uint]uint{}}
	for i := 0; uint(i) < opts.KeyCount; i++ {
		state.keyRecord[uint(i)] = 0
	}
	return WrTxn(opts, state)
}

func WrTxnWithDefaultOptsWithoutState() *MopIterator {
	return WrTxnWithoutState(DefaultWrTxnOpts())
}
