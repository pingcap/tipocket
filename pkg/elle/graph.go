//
package elle

type DiGraph struct{}

type Linear struct{}

type HistoryType uint

const (
	Invoke = iota
	Ok
)

type History struct {
	Type HistoryType
	// Can be append or read.
	Value interface{}

	Process   uint
	Index     uint
	TimeStamp uint
	Extra     interface{}
}
