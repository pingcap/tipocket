package core

import "time"

type MopValueType int
type OpType string

const (
	Invoke  OpType = "invoke"
	Ok      OpType = "ok"
	Fail    OpType = "fail"
	Info    OpType = "info"
	Nemesis OpType = "nemesis"
)

type Mop interface {
	IsAppend() bool
	IsRead() bool
}

// Append implements Mop
type Append struct {
	Key   string       `json:"key"`
	Value MopValueType `json:"value"`
}

// Read implements Mop
type Read struct {
	Key   string         `json:"key"`
	Value []MopValueType `json:"value"`
}

type Op struct {
	Index   int       `json:"index"`
	Process int       `json:"process"`
	Time    time.Time `json:"time"`
	Type    OpType    `json:"type"`
	Value   []Mop     `json:"value"`
}

type History []Op

func (a Append) IsAppend() bool {
	return true
}

func (a Append) IsRead() bool {
	return false
}

func (r Read) IsAppend() bool {
	return false
}

func (r Read) IsRead() bool {
	return true
}
