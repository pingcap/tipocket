package pessimistic

import (
	"math/rand"
	"time"
)

// randIDGenerator generates random ID that combines round-robin and zipf distribution and make sure not duplicated.
type randIDGenerator struct {
	allocated     map[uint64]struct{}
	zipf          *rand.Zipf
	uniform       *rand.Rand
	tableNum      int
	max           uint64
	partitionSize uint64
	partitionIdx  uint64
}

func (r *randIDGenerator) nextTableAndRowPairs() (fromTableID, toTableID int, fromRowID, toRowID uint64) {
	fromTableID = r.nextTableID()
	toTableID = r.nextTableID()
	fromRowID = r.nextRowID()
	toRowID = r.nextRowID()
	return
}

func (r *randIDGenerator) nextTableID() int {
	return r.uniform.Intn(r.tableNum)
}

func (r *randIDGenerator) nextRowID() uint64 {
	for {
		v := r.zipf.Uint64()
		if _, ok := r.allocated[v]; ok {
			continue
		}
		r.allocated[v] = struct{}{}
		return v
	}
}

func (r *randIDGenerator) nextNonExistRowID() uint64 {
	for {
		v := r.zipf.Uint64() + r.max
		if _, ok := r.allocated[v]; ok {
			continue
		}
		r.allocated[v] = struct{}{}
		return v
	}
}

func (r *randIDGenerator) reset() {
	r.allocated = map[uint64]struct{}{}
}

func (r *randIDGenerator) nextNumStatements(n int) int {
	return r.uniform.Intn(n)
}

func (r *randIDGenerator) nextUniqueIndex() uint64 {
	return r.uniform.Uint64()
}

func newRandIDGenerator(tableNum int, maxSize uint64) *randIDGenerator {
	src := rand.NewSource(time.Now().UnixNano())
	ran := rand.New(src)
	zipf := rand.NewZipf(ran, 1.01, 1, maxSize-1)
	return &randIDGenerator{
		allocated: map[uint64]struct{}{},
		zipf:      zipf,
		uniform:   ran,
		tableNum:  tableNum,
		max:       maxSize,
	}
}
