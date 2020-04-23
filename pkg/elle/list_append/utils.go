package list_append

import "github.com/pingcap/tipocket/pkg/elle/core"

// Note: it will check if all mops are :append and :r
//  In our current implement, it's true.
func verifyMopTypes(mop core.Mop) bool {
	return true
}

// Takes a history of txns made up of appends and reads, and checks to make
//  sure that every invoke appending a value to a key chose a unique value.
func verifyUniqueAppends(history core.History) bool {
	panic("implement me")
}

