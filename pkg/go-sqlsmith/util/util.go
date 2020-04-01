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

package util

import (
	"bytes"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
)

// BufferOut parser ast node to SQL string
func BufferOut(node ast.Node) (string, error) {
	out := new(bytes.Buffer)
	err := node.Restore(format.NewRestoreCtx(format.RestoreStringDoubleQuotes, out))
	if err != nil {
		return "", err
	}
	return string(out.Bytes()), nil
}

// TimeMustParse wrap time.Parse and panic when error
func TimeMustParse(layout, value string) time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		log.Fatalf("parse time err %+v, layout: %s, value: %s", err, layout, value)
	}
	return t
}

// MinInt ...
func MinInt(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// MaxInt ...
func MaxInt(a, b int) int {
	if a < b {
		return b
	}
	return a
}
