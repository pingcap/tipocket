// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"sync"

	"github.com/pingcap/tipocket/test-infra/pkg/core"
)

// opRecord is similar to core.Operation, but it stores data in json.RawMessage
// instead of interface{} in order to marshal into bytes.
type opRecord struct {
	Action string          `json:"action"`
	Proc   int64           `json:"proc"`
	Data   json.RawMessage `json:"data"`
}

// TODO: different operation for initial state and final state.
const dumpOperation = "dump"

// Recorder records operation history.
type Recorder struct {
	sync.Mutex
	f *os.File
}

// NewRecorder creates a recorder to log the history to the file.
func NewRecorder(name string) (*Recorder, error) {
	os.MkdirAll(path.Dir(name), 0755)

	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	return &Recorder{f: f}, nil
}

// Close closes the recorder.
func (r *Recorder) Close() {
	r.f.Close()
}

// RecordState records the request.
func (r *Recorder) RecordState(state interface{}) error {
	return r.record(0, dumpOperation, state)
}

// RecordRequest records the request.
func (r *Recorder) RecordRequest(proc int64, op interface{}) error {
	return r.record(proc, core.InvokeOperation, op)
}

// RecordResponse records the response.
func (r *Recorder) RecordResponse(proc int64, op interface{}) error {
	return r.record(proc, core.ReturnOperation, op)
}

func (r *Recorder) record(proc int64, action string, op interface{}) error {
	// Marshal the op to json in order to store it in a history file.
	data, err := json.Marshal(op)
	if err != nil {
		return err
	}

	v := opRecord{
		Action: action,
		Proc:   proc,
		Data:   json.RawMessage(data),
	}

	data, err = json.Marshal(v)
	if err != nil {
		return err
	}

	r.Lock()
	defer r.Unlock()

	if _, err = r.f.Write(data); err != nil {
		return err
	}

	if _, err = r.f.WriteString("\n"); err != nil {
		return err
	}

	return nil
}

// RecordParser is to parses the operation data.
// It must be thread-safe.
type RecordParser interface {
	// OnRequest parses an operation data to model's input.
	OnRequest(data json.RawMessage) (interface{}, error)
	// OnResponse parses an operation data to model's output.
	// Return nil means the operation has an infinite end time.
	// E.g, we meet timeout for a operation.
	OnResponse(data json.RawMessage) (interface{}, error)
	// If we have some infinite operations, we should return a
	// noop response to complete the operation.
	OnNoopResponse() interface{}
	// OnState parses model state json data to model's state
	OnState(state json.RawMessage) (interface{}, error)
}

// ReadHistory reads operations and a model state from a history file.
func ReadHistory(historyFile string, p RecordParser) ([]core.Operation, interface{}, error) {
	f, err := os.Open(historyFile)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var state interface{}
	ops := make([]core.Operation, 0, 1024)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var record opRecord
		if err = json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, nil, err
		}

		var data interface{}
		if record.Action == core.InvokeOperation {
			if data, err = p.OnRequest(record.Data); err != nil {
				return nil, nil, err
			}
		} else if record.Action == core.ReturnOperation {
			if data, err = p.OnResponse(record.Data); err != nil {
				return nil, nil, err
			}
		} else {
			if state, err = p.OnState(record.Data); err != nil {
				return nil, nil, err
			}
			// A dumped state is not an operation.
			continue
		}

		op := core.Operation{
			Action: record.Action,
			Proc:   record.Proc,
			Data:   data,
		}
		ops = append(ops, op)
	}

	if err = scanner.Err(); err != nil {
		return nil, nil, err
	}

	return ops, state, nil
}

// int64Slice attaches the methods of Interface to []int, sorting in increasing order.
type int64Slice []int64

func (p int64Slice) Len() int           { return len(p) }
func (p int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// CompleteOperations completes the history of operation.
func CompleteOperations(ops []core.Operation, p RecordParser) ([]core.Operation, error) {
	procID := map[int64]struct{}{}
	compOps := make([]core.Operation, 0, len(ops))
	for _, op := range ops {
		if op.Action == core.InvokeOperation {
			if _, ok := procID[op.Proc]; ok {
				return nil, fmt.Errorf("missing return, op: %v", op)
			}
			procID[op.Proc] = struct{}{}
			compOps = append(compOps, op)
		} else {
			if _, ok := procID[op.Proc]; !ok {
				return nil, fmt.Errorf("missing invoke, op: %v", op)
			}
			if op.Data == nil {
				continue
			}
			delete(procID, op.Proc)
			compOps = append(compOps, op)
		}
	}

	// To get a determined complete history of operations, we sort procIDs.
	var keys []int64
	for k := range procID {
		keys = append(keys, k)
	}
	sort.Sort(int64Slice(keys))

	for _, proc := range keys {
		op := core.Operation{
			Action: core.ReturnOperation,
			Proc:   proc,
			Data:   p.OnNoopResponse(),
		}
		compOps = append(compOps, op)
	}

	return compOps, nil
}
