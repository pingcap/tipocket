package txn

import (
	"testing"

	"github.com/mohae/deepcopy"
)

func TestWrTxn(t *testing.T) {
	iter := WrTxnWithDefaultOpts()
	keyRecord := map[string]int{}
	for loopCnt := 0; loopCnt < 1000; loopCnt++ {
		mops := iter.Next()
		if len(mops) == 0 {
			t.Fatal("iter.Next contains no data")
		}

		for _, mop := range mops {
			if mop.IsAppend() {
				lastVersion, e := keyRecord[mop.GetKey()]
				if e == false {
					keyRecord[mop.GetKey()] = 1
					lastVersion = 1
				}
				aVersion := mop.GetValue().(int)
				if lastVersion != aVersion {
					t.Fatalf("Version of append is wrong, expected %d, got %d", aVersion, lastVersion)
				}
				keyRecord[mop.GetKey()] = keyRecord[mop.GetKey()] + 1
			}
		}
		dataMap := deepcopy.Copy(iter.activeKeys.keyRecord)
		keyRecord = dataMap.(map[string]int)
	}
}
