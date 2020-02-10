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

package model

import (
	"encoding/json"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/history"
)

// Op is an operation.
type Op bool

const (
	// RegisterRead a register
	RegisterRead Op = false
	// RegisterWrite a register
	RegisterWrite Op = true
)

// RegisterRequest is the request that is issued to a register.
type RegisterRequest struct {
	Op    Op
	Value int
}

// RegisterResponse is the response returned by a register.
type RegisterResponse struct {
	Unknown bool
	Value   int
}

var _ core.UnknownResponse = (*RegisterResponse)(nil)

// IsUnknown implements UnknownResponse interface
func (r RegisterResponse) IsUnknown() bool {
	return r.Unknown
}

type register struct {
	perparedState *int
}

func (r *register) Prepare(state interface{}) {
	s := state.(int)
	r.perparedState = &s
}

func (r *register) Init() interface{} {
	if r.perparedState != nil {
		return *r.perparedState
	}
	return 0
}

func (*register) Step(state interface{}, input interface{}, output interface{}) (bool, interface{}) {
	st := state.(int)
	inp := input.(RegisterRequest)
	out := output.(RegisterResponse)

	// read
	if inp.Op == RegisterRead {
		ok := out.Value == st || out.Unknown
		return ok, st
	}

	// write
	return true, inp.Value
}

func (*register) Equal(state1, state2 interface{}) bool {
	st1 := state1.(int)
	st2 := state2.(int)
	return st1 == st2
}

func (*register) Name() string {
	return "register"
}

// RegisterModel returns a read/write register model
func RegisterModel() core.Model {
	return &register{}
}

type registerParser struct {
}

func (p registerParser) OnRequest(data json.RawMessage) (interface{}, error) {
	r := RegisterRequest{}
	err := json.Unmarshal(data, &r)
	return r, err
}

func (p registerParser) OnResponse(data json.RawMessage) (interface{}, error) {
	r := RegisterResponse{}
	err := json.Unmarshal(data, &r)
	if r.Unknown {
		return nil, err
	}
	return r, nil
}

func (p registerParser) OnNoopResponse() interface{} {
	return RegisterResponse{Unknown: true}
}

func (p registerParser) OnState(data json.RawMessage) (interface{}, error) {
	var state int
	err := json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

// RegisterParser parses Register history.
func RegisterParser() history.RecordParser {
	return registerParser{}
}
