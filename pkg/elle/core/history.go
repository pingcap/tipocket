package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/juju/errors"
)

var (
	operationPattern = regexp.MustCompile(`\{(.*)\}`)
	opIndexPattern   = regexp.MustCompile(`:index\s+([0-9]+)`)
	opTimePattern    = regexp.MustCompile(`:time\s+([0-9]+)`)
	opProcessPattern = regexp.MustCompile(`:process\s+([0-9]+|:nemesis)`)
	opTypePattern    = regexp.MustCompile(`:type\s+(:[a-zA-Z]+)`)
	opValuePattern   = regexp.MustCompile(`:value\s+\[(.*)\]`)
	mopPattern       = regexp.MustCompile(`(\[:(append|r)\s+(\d+)\s+(\[.*?\]|.*?)\])+`)
	mopValuePattern  = regexp.MustCompile(`\[(.*)\]`)
)

const NemesisProcessMagicNumber = -1

// MopValueType ...
type MopValueType interface{}

// OpType ...
type OpType string

// MopType ...
type MopType string

// OpType enums
const (
	OpTypeInvoke OpType = "invoke"
	OpTypeOk     OpType = "ok"
	OpTypeFail   OpType = "fail"
	OpTypeInfo   OpType = "info"
)

// MopType enums
const (
	MopTypeAll    MopType = "all"
	MopTypeAppend MopType = "append"
	MopTypeRead   MopType = "read"
)

// Mop interface
type Mop interface {
	fmt.Stringer

	IsAppend() bool
	IsRead() bool
	GetMopType() MopType
	GetKey() string
	GetValue() MopValueType
	IsEqual(b Mop) bool
}

// Append implements Mop
type Append struct {
	Key   string       `json:"key"`
	Value MopValueType `json:"value"`
}

func (a Append) String() string {
	return fmt.Sprintf("[:append %s %v]", a.Key, a.Value)
}

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

func (a Append) IsEqual(b Mop) bool {
	if a.GetMopType() != b.GetMopType() {
		return false
	}
	return a == b
}

// Read implements Mop
type Read struct {
	Key   string       `json:"key"`
	Value MopValueType `json:"value"`
}

func (r Read) String() string {
	return fmt.Sprintf("[:r %s %v]", r.Key, r.Value)
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

func (r Read) IsEqual(b Mop) bool {
	if r.GetMopType() != b.GetMopType() {
		return false
	}
	aValue := r.GetValue().([]int)
	bValue := r.GetValue().([]int)
	if len(aValue) != len(bValue) {
		return false
	}
	for idx := range aValue {
		if aValue[idx] != bValue[idx] {
			return false
		}
	}
	return true
}

// Op is operation
type Op struct {
	Index   IntOptional `json:"index"`
	Process IntOptional `json:"process"`
	Time    time.Time   `json:"time"`
	Type    OpType      `json:"type"`
	Value   *[]Mop      `json:"value"`
}

func (op Op) String() string {
	return fmt.Sprintf("{:type %s :process %d :time %d :index %d}", string(op.Type), op.Process.MustGet(), op.Time.Nanosecond(), op.Index.MustGet())
}

func (op Op) ValueLength() int {
	if op.Value == nil {
		return 0
	}
	return len(*op.Value)
}

// History contains operations
type History []Op

type SameKeyOpsByLength [][]MopValueType

func (b SameKeyOpsByLength) Len() int {
	return len(b)
}

func (b SameKeyOpsByLength) Less(i, j int) bool {
	return len(b[i]) < len(b[j])
}

func (b SameKeyOpsByLength) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func AllTypesHistory(history History, tp OpType) History {
	var resp History
	for _, v := range history {
		if v.Type == tp {
			resp = append(resp, v)
		}
	}
	return resp
}

// AttachIndexIfNoExists add the index for history with it's number in array.
func (h History) AttachIndexIfNoExists() {
	if len(h) != 0 && h[0].Index.Present() {
		return
	}
	for i := range h {
		h[i].Index = IntOptional{i}
	}
}

// ParseHistory parse history from elle's row text
func ParseHistory(content string) (History, error) {
	var history History
	for _, line := range strings.Split(content, "\n") {
		if line == "" {
			continue
		}
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
		op.Index = IntOptional{opIndex}
	}

	opTimeMatch := opTimePattern.FindStringSubmatch(operationMatch[1])
	if len(opTimeMatch) == 2 {
		opTime, err := strconv.Atoi(strings.Trim(opTimeMatch[1], " "))
		if err != nil {
			return empty, err
		}
		op.Time = time.Unix(0, int64(opTime))
	}

	opProcessMatch := opProcessPattern.FindStringSubmatch(operationMatch[1])
	if len(opProcessMatch) == 2 {
		if opProcessMatch[1] == ":nemesis" {
			op.Process.Set(NemesisProcessMagicNumber)
		} else {
			opProcess, err := strconv.Atoi(strings.Trim(opProcessMatch[1], " "))
			if err != nil {
				return empty, err
			}
			op.Process.Set(opProcess)
		}
	}

	opTypeMatch := opTypePattern.FindStringSubmatch(operationMatch[1])
	if len(opTypeMatch) != 2 {
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
	default:
		return empty, errors.Errorf("invalid type, %s", opTypeMatch[1])
	}

	opValueMatch := opValuePattern.FindStringSubmatch(operationMatch[1])
	// can values be empty?
	if len(opValueMatch) == 2 {
		mopContent := strings.Trim(opValueMatch[1], " ")
		if mopContent != "" {
			mopMatches := mopPattern.FindAllStringSubmatch(mopContent, -1)
			for _, mopMatch := range mopMatches {
				if len(mopMatch) != 5 {
					break
				}
				key := strings.Trim(mopMatch[3], " ")
				var value MopValueType
				mopValueMatches := mopValuePattern.FindStringSubmatch(mopMatch[4])
				if len(mopValueMatches) == 2 {
					values := []int{}
					trimVal := strings.Trim(mopValueMatches[1], "[")
					trimVal = strings.Trim(trimVal, "]")
					if trimVal != "" {
						for _, valStr := range strings.Split(trimVal, " ") {
							val, err := strconv.Atoi(valStr)
							if err != nil {
								return empty, err
							}
							values = append(values, val)
						}
					}
					value = values
				} else {
					trimVal := strings.Trim(mopMatch[4], " ")
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
				switch mopMatch[2] {
				case "append":
					mop = Append{Key: key, Value: value}
				case "r":
					mop = Read{Key: key, Value: value}
				default:
					panic("unreachable")
				}
				if op.Value == nil {
					destArray := make([]Mop, 0)
					op.Value = &destArray
				}
				*op.Value = append(*op.Value, mop)
			}
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
		if op.Process.Present() && op.Process.MustGet() == p {
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
		for _, mop := range *op.Value {
			if t == MopTypeAll || mop.GetMopType() == t {
				keys = append(keys, mop.GetKey())
			}
		}
	}

	return keys
}
