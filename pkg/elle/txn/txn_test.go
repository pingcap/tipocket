package txn

import (
	"testing"

	"github.com/mohae/deepcopy"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

func TestWrTxn(t *testing.T) {
	iter := WrTxnWithDefaultOptsWithoutState()
	keyRecord := map[string]uint{}
	for loopCnt := 0; loopCnt < 10; loopCnt++ {
		mops := iter.Next()
		if len(mops) == 0 {
			t.Fatal("iter.Next contains no data")
		}

		for _, v := range mops {
			if v.IsAppend() {
				appendRecord := v.(core.Append)

				lastVersion := keyRecord[appendRecord.Key]
				aVersion := appendRecord.Value.(uint)
				if lastVersion != aVersion {
					t.Fatalf("Version of append is wrong, expected %d, got %d", aVersion, lastVersion)
				}
				keyRecord[appendRecord.Key] = keyRecord[appendRecord.Key] + 1
			} else {
				_ = v.(core.Read)
				//if readRecord.Value == nil {
				//	t.Fatal("length of record.Value is not zero")
				//}
			}
		}
		dataMap := deepcopy.Copy(iter.activeKeys.keyRecord)
		keyRecord = dataMap.(map[string]uint)
	}
}
