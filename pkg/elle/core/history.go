package core

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/juju/errors"
)

// {:index 0 :type :invoke  :value [[:append 253 1] [:append 253 3] [:append 253 4] [:append 255 2] [:append 255 3] [:append 255 4] [:append 255 5] [:append 256 1] [:append 256 2]]}
// {:index 1 :type :ok      :value [[:append 253 1] [:append 253 3] [:append 253 4] [:append 255 2] [:append 255 3] [:append 255 4] [:append 255 5] [:append 256 1] [:append 256 2]]}
// {:index 7 :type :ok      :value [[:append 250 10] [:r 253 [1 3 4]] [:r 255 [2 3 4 5]] [:append 256 3]]}

var (
	operationPattern = regexp.MustCompile(`\{(.*)\}`)
	opIndexPattern   = regexp.MustCompile(`:index\s+(.*?)\s?:`)
	opTypePattern    = regexp.MustCompile(`:type\s+(:?.*?),?\s+:?`)
	opValuePattern   = regexp.MustCompile(`:value\s+\[(.*)\]`)
	mopPattern       = regexp.MustCompile(`\[:(append|r)\s+(\d+)\s+(\[.*?\]|.*?)\](.*)`)
	mopValuePattern  = regexp.MustCompile(`\[(.*)\]`)
)

type MopValueType interface{}
type OpType string

// OpType enums
const (
	Invoke  OpType = "invoke"
	Ok      OpType = "ok"
	Fail    OpType = "fail"
	Info    OpType = "info"
	Nemesis OpType = "nemesis"
)

// Mop interface
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
	Key   string       `json:"key"`
	Value MopValueType `json:"value"`
}

// Op is operation
type Op struct {
	Index   int       `json:"index"`
	Process int       `json:"process"`
	Time    time.Time `json:"time"`
	Type    OpType    `json:"type"`
	Value   []Mop     `json:"value"`
}

// History contains operations
type History []Op

// IsAppend ...
func (a Append) IsAppend() bool {
	return true
}

// IsRead ...
func (a Append) IsRead() bool {
	return false
}

// IsAppend ...
func (r Read) IsAppend() bool {
	return false
}

// IsRead ...
func (r Read) IsRead() bool {
	return true
}

// ParseHistory parse history from elle's row text
func ParseHistory(content string) (History, error) {
	var history History
	for _, line := range strings.Split(content, "\n") {
		op, err := ParseOp(strings.Trim(line, " "))
		if err != nil {
			return nil, err
		}
		history = append(history, op)
	}
	return history, nil
}

// ParseOp parse operation from elle's row text
// TODO: parse process and time field (they are optional)
func ParseOp(opString string) (Op, error) {
	var (
		empty Op
		op    Op
	)
	operationMatch := operationPattern.FindStringSubmatch(opString)
	if len(operationMatch) != 2 {
		return empty, errors.New("operation should surrounded by {}")
	}

	opIndexMatch := opIndexPattern.FindStringSubmatch(operationMatch[1])
	if len(opIndexMatch) != 2 {
		return empty, errors.New("operation should have :index field")
	}
	opIndex, err := strconv.Atoi(strings.Trim(opIndexMatch[1], " "))
	if err != nil {
		return empty, err
	}
	op.Index = opIndex

	opTypeMatch := opTypePattern.FindStringSubmatch(operationMatch[1])
	if len(opTypeMatch) != 2 {
		return empty, errors.New("operation should have :type field")
	}
	switch opTypeMatch[1] {
	case ":invoke":
		op.Type = Invoke
	case ":ok":
		op.Type = Ok
	case ":fail":
		op.Type = Fail
	case ":info":
		op.Type = Info
	case ":nemesis":
		op.Type = Nemesis
	default:
		return empty, errors.Errorf("invalid type, %s", opTypeMatch[1])
	}

	opValueMatch := opValuePattern.FindStringSubmatch(operationMatch[1])
	// can values be empty?
	if len(opValueMatch) != 2 {
		return empty, errors.New("operation should have :value field")
	}
	mopContent := strings.Trim(opValueMatch[1], " ")
	for mopContent != "" {
		mopMatch := mopPattern.FindStringSubmatch(mopContent)
		if len(mopMatch) != 5 {
			break
		}
		mopContent = strings.Trim(mopMatch[4], " ")

		key := strings.Trim(mopMatch[2], " ")
		var value MopValueType
		mopValueMatches := mopValuePattern.FindStringSubmatch(mopMatch[3])
		if len(mopValueMatches) == 2 {
			values := []int{}
			trimVal := strings.Trim(mopValueMatches[1], "[")
			trimVal = strings.Trim(trimVal, "]")
			for _, valStr := range strings.Split(trimVal, " ") {
				val, err := strconv.Atoi(valStr)
				if err != nil {
					return empty, err
				}
				values = append(values, val)
			}
			value = values
		} else {
			trimVal := strings.Trim(mopMatch[3], " ")
			if trimVal == "nil" {
				value = nil
			} else {
				val, err := strconv.Atoi(trimVal)
				if err != nil {
					return empty, err
				}
				value = val
			}
		}

		var mop Mop
		switch mopMatch[1] {
		case "append":
			mop = Append{Key: key, Value: value}
		case "r":
			mop = Read{Key: key, Value: value}
		default:
			panic("unreachable")
		}
		op.Value = append(op.Value, mop)
	}

	return op, nil
}
