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
	"encoding/json"
)

// NoopRequest is a noop request.
type NoopRequest struct {
	// 0 for read, 1 for write
	Op    int
	Value int
}

// NoopResponse is a noop response.
type NoopResponse struct {
	Value   int
	Ok      bool
	Unknown bool
}

type action struct {
	proc int64
	op   interface{}
}

// NoopParser is a noop parser.
type NoopParser struct {
	State int
}

// OnRequest impls RecordParser.
func (p NoopParser) OnRequest(data json.RawMessage) (interface{}, error) {
	r := NoopRequest{}
	err := json.Unmarshal(data, &r)
	return r, err
}

// OnResponse impls RecordParser.
func (p NoopParser) OnResponse(data json.RawMessage) (interface{}, error) {
	r := NoopResponse{}
	err := json.Unmarshal(data, &r)
	if r.Unknown {
		return nil, err
	}
	return r, err
}

// OnNoopResponse impls RecordParser.
func (p NoopParser) OnNoopResponse() interface{} {
	return NoopResponse{Unknown: true}
}

// OnState impls RecordParser.
func (p NoopParser) OnState(state json.RawMessage) (interface{}, error) {
	return p.State, nil
}
