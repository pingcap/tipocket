package types

import (
	"time"
)

// Log line struct
type Log struct {
	Time  time.Time
	Node  int
	SQL   *SQL
	State string
}

// ByLog implement sort interface for log
type ByLog []*Log

func (a ByLog) Len() int           { return len(a) }
func (a ByLog) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLog) Less(i, j int) bool {
	return a[i].GetTime().Before(a[j].GetTime())
}

// GetTime get log time
func (l *Log) GetTime() time.Time {
	return l.Time
}

// GetNode get executor id
func (l *Log) GetNode() int {
	return l.Node
}

// GetSQL get SQL struct
func (l *Log) GetSQL() *SQL {
	if l.SQL != nil {
		return l.SQL
	}
	return &SQL{}
}
