package txn

import (
	"testing"

	"github.com/mohae/deepcopy"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

func mustAtou(s string) uint {
	return uint(mustAtoi(s))
}

func TestWrTxn(t *testing.T) {
	iter := WrTxnWithDefaultOptsWithoutState()
	keyRecord := map[uint]uint{}
	for loopCnt := 0; loopCnt < 10; loopCnt++ {
		mops := iter.Next()
		if len(mops) == 0 {
			t.Fatal("iter.Next contains no data")
		}

		for _, v := range mops {
			if v.IsAppend() {
				appendRecord := v.(core.Append)

				lastVersion := keyRecord[mustAtou(appendRecord.Key)]
				aVersion := appendRecord.Value
				if lastVersion != uint(aVersion) {
					t.Fatalf("Version of append is wrong, expected %d, got %d", aVersion, lastVersion)
				}
				keyRecord[mustAtou(appendRecord.Key)] = keyRecord[mustAtou(appendRecord.Key)] + 1
			} else {
				readRecord := v.(core.Read)
				if len(readRecord.Value) != 0 {
					t.Fatal("length of record.Value is not zero")
				}
			}
		}
		dataMap := deepcopy.Copy(iter.activeKeys.keyRecord)
		keyRecord = dataMap.(map[uint]uint)
	}
}
