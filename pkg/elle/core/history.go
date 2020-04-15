package core

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/juju/errors"
)

var (
	operationPattern = regexp.MustCompile(`\{(.*)\}`)
	opIndexPattern   = regexp.MustCompile(`:index\s+(.*?)\s?:`)
	opProcessPattern = regexp.MustCompile(`:process\s+(.*?)\s?:`)
	opTypePattern    = regexp.MustCompile(`:type\s+(:?.*?)(,?\s+:?|$)`)
	opValuePattern   = regexp.MustCompile(`:value\s+\[(.*)\]`)
	mopPattern       = regexp.MustCompile(`\[:(append|r)\s+(\d+)\s+(\[.*?\]|.*?)\](.*)`)
	mopValuePattern  = regexp.MustCompile(`\[(.*)\]`)
)

// MopValueType ...
type MopValueType interface{}

// OpType ...
type OpType string

// MopType ...
type MopType string

// OpType enums
const (
	OpTypeInvoke  OpType = "invoke"
	OpTypeOk      OpType = "ok"
	OpTypeFail    OpType = "fail"
	OpTypeInfo    OpType = "info"
	OpTypeNemesis OpType = "nemesis"
)

// MopType enums
const (
	MopTypeAll    MopType = "all"
	MopTypeAppend MopType = "append"
	MopTypeRead   MopType = "read"
)

// Mop interface
type Mop interface {
	IsAppend() bool
	IsRead() bool
	GetMopType() MopType
	GetKey() string
	GetValue() MopValueType
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
func (Append) IsAppend() bool {
	return true
}

// IsRead ...
func (Append) IsRead() bool {
	return false
}

// GetMopType ...
func (Append) GetMopType() MopType {
	return MopTypeAppend
}

// GetKey get append key
func (a Append) GetKey() string {
	return a.Key
}

// GetValue get append value
func (a Append) GetValue() MopValueType {
	return a.Value
}

// IsAppend ...
func (Read) IsAppend() bool {
	return false
}

// IsRead ...
func (Read) IsRead() bool {
	return true
}

// GetMopType ...
func (Read) GetMopType() MopType {
	return MopTypeRead
}

// GetKey get read key
func (r Read) GetKey() string {
	return r.Key
}

// GetValue get read value
func (r Read) GetValue() MopValueType {
	return r.Value
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
	if len(opIndexMatch) == 2 {
		opIndex, err := strconv.Atoi(strings.Trim(opIndexMatch[1], " "))
		if err != nil {
			return empty, err
		}
		op.Index = opIndex
	}

	opProcessMatch := opProcessPattern.FindStringSubmatch(operationMatch[1])
	if len(opProcessMatch) == 2 {
		opProcess, err := strconv.Atoi(strings.Trim(opProcessMatch[1], " "))
		if err != nil {
			return empty, err
		}
		op.Process = opProcess
	}

	opTypeMatch := opTypePattern.FindStringSubmatch(operationMatch[1])
	if len(opTypeMatch) != 3 {
		return empty, errors.New("operation should have :type field")
	}
	switch opTypeMatch[1] {
	case ":invoke":
		op.Type = OpTypeInvoke
	case ":ok":
		op.Type = OpTypeOk
	case ":fail":
		op.Type = OpTypeFail
	case ":info":
		op.Type = OpTypeInfo
	case ":nemesis":
		op.Type = OpTypeNemesis
	default:
		return empty, errors.Errorf("invalid type, %s", opTypeMatch[1])
	}

	opValueMatch := opValuePattern.FindStringSubmatch(operationMatch[1])
	// can values be empty?
	if len(opValueMatch) == 2 {
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
	}

	return op, nil
}

// FilterType filter by type
func (h History) FilterType(t OpType) History {
	var filterHistory History
	for _, op := range h {
		if op.Type == t {
			filterHistory = append(filterHistory, op)
		}
	}
	return filterHistory
}

// FilterProcess filter by process
func (h History) FilterProcess(p int) History {
	var filterHistory History
	for _, op := range h {
		if op.Process == p {
			filterHistory = append(filterHistory, op)
		}
	}
	return filterHistory
}

// GetKeys get keys by a given type
// MopTypeAll to get keys of all types
func (h History) GetKeys(t MopType) []string {
	var keys []string

	for _, op := range h {
		for _, mop := range op.Value {
			if t == MopTypeAll || mop.GetMopType() == t {
				keys = append(keys, mop.GetKey())
			}
		}
	}

	return keys
}
