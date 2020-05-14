package core

import (
	"fmt"
	"reflect"
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
	mopPattern       = regexp.MustCompile(`(\[:(append|r)\s+(\w+)\s+(\[.*?\]|.*?)\])+`)
	mopValuePattern  = regexp.MustCompile(`\[(.*)\]`)
)

// NemesisProcessMagicNumber is a magic number to stand for nemesis related event on a history
const NemesisProcessMagicNumber = -1

// AnonymousMagicNumber is a magic number, means that we don't know the process id of a history event
const AnonymousMagicNumber = -2

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
type Mop struct {
	T MopType                `json:"type"`
	M map[string]interface{} `json:"m"`
}

// IsAppend ...
func (m Mop) IsAppend() bool {
	return m.T == MopTypeAppend
}

// IsRead ...
func (m Mop) IsRead() bool {
	return m.T == MopTypeRead
}

// GetMopType ...
func (m Mop) GetMopType() MopType {
	return m.T
}

// GetKey ...
func (m Mop) GetKey() string {
	return m.M["key"].(string)
}

// GetValue ...
func (m Mop) GetValue() MopValueType {
	value := m.M["value"]
	if value == nil {
		return nil
	}
	return value.(MopValueType)
}

// IsEqual ...
func (m Mop) IsEqual(n Mop) bool {
	return reflect.DeepEqual(m, n)
}

// String ...
func (m Mop) String() string {
	switch m.T {
	case MopTypeRead:
		key := m.M["key"].(string)
		value := m.M["value"]
		if value == nil {
			return fmt.Sprintf("[:r %s nil]", key)
		}
		return fmt.Sprintf("[:r %s %v]", key, value)
	case MopTypeAppend:
		key := m.M["key"].(string)
		value := m.M["value"].(int)
		return fmt.Sprintf("[:append %s %v]", key, value)
	default:
		panic("unreachable")
	}
}

// Append implements Mop
func Append(key string, value int) Mop {
	return Mop{
		T: MopTypeAppend,
		M: map[string]interface{}{
			"key":   key,
			"value": value,
		},
	}
}

// Read ...
func Read(key string, values []int) Mop {
	if values == nil {
		return Mop{
			T: MopTypeRead,
			M: map[string]interface{}{
				"key":   key,
				"value": nil,
			},
		}
	}
	return Mop{
		T: MopTypeRead,
		M: map[string]interface{}{
			"key":   key,
			"value": values,
		},
	}
}

// Op is operation
type Op struct {
	Index   IntOptional `json:"index,omitempty"`
	Process IntOptional `json:"process,omitempty"`
	Time    time.Time   `json:"time"`
	Type    OpType      `json:"type"`
	Value   *[]Mop      `json:"value"`
	Error   string      `json:"error,omitempty"`
}

func (op Op) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("{:type :%s", op.Type))
	var mopParts []string
	if op.Value != nil {
		for _, mop := range *op.Value {
			mopParts = append(mopParts, mop.String())
		}
	}
	parts = append(parts, fmt.Sprintf(":value [%s]", strings.Join(mopParts, " ")))
	if op.Process.Present() {
		parts = append(parts, fmt.Sprintf(":process %d", op.Process.MustGet()))
	}
	if !op.Time.IsZero() {
		parts = append(parts, fmt.Sprintf(":time %d", op.Time.UnixNano()))
	}
	if op.Index.Present() {
		parts = append(parts, fmt.Sprintf(":index %d", op.Index.MustGet()))
	}

	if op.Error != "" {
		parts = append(parts, fmt.Sprintf(":error [\"%s\"]", op.Error))
	}
	return strings.Join(parts, ", ") + "}"
}

// ValueLength ...
func (op Op) ValueLength() int {
	if op.Value == nil {
		return 0
	}
	return len(*op.Value)
}

// History contains operations
type History []Op

// SameKeyOpsByLength ...
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
					mop = Append(key, value.(int))
				case "r":
					if value != nil {
						mop = Read(key, value.([]int))
					} else {
						mop = Read(key, nil)
					}
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
