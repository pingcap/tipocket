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
	"testing"

	"github.com/anishathalye/porcupine"

	"github.com/pingcap/tipocket/test-infra/pkg/core"
)

func convertModel(m core.Model) porcupine.Model {
	return porcupine.Model{
		Init:  m.Init,
		Step:  m.Step,
		Equal: m.Equal,
	}
}

func TestCasRegisterModel(t *testing.T) {
	events := []porcupine.Event{
		{Kind: porcupine.CallEvent, Value: CasRegisterRequest{Op: CasRegisterWrite, Arg1: 100, Arg2: 0}, Id: 0},
		{Kind: porcupine.CallEvent, Value: CasRegisterRequest{Op: CasRegisterRead}, Id: 1},
		{Kind: porcupine.CallEvent, Value: CasRegisterRequest{Op: CasRegisterRead}, Id: 2},
		{Kind: porcupine.CallEvent, Value: CasRegisterRequest{Op: CasRegisterCAS, Arg1: 100, Arg2: 200}, Id: 3},
		{Kind: porcupine.CallEvent, Value: CasRegisterRequest{Op: CasRegisterRead}, Id: 4},
		{Kind: porcupine.ReturnEvent, Value: CasRegisterResponse{}, Id: 2},
		{Kind: porcupine.ReturnEvent, Value: CasRegisterResponse{Value: 100}, Id: 1},
		{Kind: porcupine.ReturnEvent, Value: CasRegisterResponse{}, Id: 0},
		{Kind: porcupine.ReturnEvent, Value: CasRegisterResponse{Ok: true}, Id: 3},
		{Kind: porcupine.ReturnEvent, Value: CasRegisterResponse{Value: 200}, Id: 4},
	}
	res := porcupine.CheckEvents(convertModel(CasRegisterModel()), events)
	if res != true {
		t.Fatal("expected operations to be linearizable")
	}
}

func TestCasRegisterModelPrepare(t *testing.T) {
	model := RegisterModel()
	model.Prepare(888)
	state := model.Init()
	if state.(int) != 888 {
		t.Fatalf("expected to be 888, got %v", state)
	}
}
