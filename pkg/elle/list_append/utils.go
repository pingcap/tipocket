package list_append

import (
	"github.com/pingcap/tipocket/pkg/elle/core"
)

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

// Takes a map of keys to observed values (e.g. from
//  `sorted-values`, and verifies that for each key, the values
//  read are consistent with a total order of appends. For instance, these values
//  are consistent:
// Returns nil if the history is OK, or throws otherwise.
func verifyTotalOrders(history core.History) bool {
	panic("implement me")
}
