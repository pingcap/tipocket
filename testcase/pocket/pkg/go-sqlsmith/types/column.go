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

package types

import "github.com/pingcap/parser/ast"

// Column defines database column
type Column struct {
	DB           string
	Table        string
	OriginTable  string
	Column       string
	OriginColumn string
	DataType     string
	DataLen      int
	Func         bool
	NewFunc      bool
	Options      []ast.ColumnOptionType
}

// Clone makes a replica of column
func (c *Column) Clone() *Column {
	return &Column{
		DB:           c.DB,
		Table:        c.Table,
		OriginTable:  c.OriginTable,
		Column:       c.Column,
		OriginColumn: c.OriginColumn,
		DataType:     c.DataType,
		DataLen:      c.DataLen,
		Func:         c.Func,
		NewFunc:      c.NewFunc,
		Options:      c.Options,
	}
}

// AddOption add option for column
func (c *Column) AddOption(opt ast.ColumnOptionType) {
	for _, option := range c.Options {
		if option == opt {
			return
		}
	}
	c.Options = append(c.Options, opt)
}

// HasOption return is has the given option
func (c *Column) HasOption(opt ast.ColumnOptionType) bool {
	for _, option := range c.Options {
		if option == opt {
			return true
		}
	}
	return false
}
