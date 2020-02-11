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

package verify

import (
	"log"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/history"
)

// Suit collects a checker, a model and a parser.
type Suit struct {
	Checker core.Checker
	Model   core.Model
	Parser  history.RecordParser
}

// Verify creates the verifier from model name and verfies the history file.
func (s Suit) Verify(historyFile string) {
	if s.Model == nil {
		log.Printf("begin to check %s", s.Checker.Name())
	} else {
		log.Printf("begin to check %s with %s", s.Model.Name(), s.Checker.Name())
	}
	ops, state, err := history.ReadHistory(historyFile, s.Parser)
	if err != nil {
		log.Fatalf("verify failed: %v", err)
	}

	ops, err = history.CompleteOperations(ops, s.Parser)
	if err != nil {
		log.Fatalf("verify failed: %v", err)
	}

	if s.Model != nil {
		s.Model.Prepare(state)
	}
	ok, err := s.Checker.Check(s.Model, ops)
	if err != nil {
		log.Fatalf("verify history failed %v", err)
	}

	if !ok {
		log.Fatalf("history %s is not valid", historyFile)
	} else {
		log.Printf("history %s is valid", historyFile)
	}
}
