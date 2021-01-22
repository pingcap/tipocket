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

package builtin

import "github.com/pingcap/parser/ast"

var logicFunctions = []*functionClass{
	{ast.LogicAnd, 2, 2, false, true, false},
	{ast.LogicOr, 2, 2, false, true, false},
	{ast.LogicXor, 2, 2, false, true, false},
	{ast.GE, 2, 2, false, true, false},
	{ast.LE, 2, 2, false, true, false},
	{ast.EQ, 2, 2, false, true, false},
	{ast.NE, 2, 2, false, true, false},
	{ast.LT, 2, 2, false, true, false},
	{ast.GT, 2, 2, false, true, false},
	{ast.NullEQ, 2, 2, false, true, false},
	{ast.Plus, 2, 2, false, true, false},
	{ast.Minus, 2, 2, false, true, false},
	{ast.Mod, 2, 2, false, true, false},
	{ast.Div, 2, 2, false, true, false},
	{ast.Mul, 2, 2, false, true, false},
	{ast.IntDiv, 2, 2, false, true, false},
	{ast.BitNeg, 1, 1, false, true, false},
	{ast.And, 2, 2, false, true, false},
	{ast.LeftShift, 2, 2, false, true, false},
	{ast.RightShift, 2, 2, false, true, false},
	{ast.UnaryNot, 1, 1, false, true, false},
	{ast.Or, 2, 2, false, true, false},
	{ast.Xor, 2, 2, false, true, false},
	{ast.UnaryMinus, 1, 1, false, true, false},
	{ast.In, 2, -1, false, true, false},
	{ast.IsTruth, 1, 1, false, true, false},
	{ast.IsFalsity, 1, 1, false, true, false},
	{ast.Like, 3, 3, false, true, false},
	{ast.Regexp, 2, 2, false, true, false},
	{ast.Case, 1, -1, false, true, false},
	{ast.RowFunc, 2, -1, false, true, false},
	{ast.SetVar, 2, 2, false, true, false},
	{ast.GetVar, 1, 1, false, true, false},
	{ast.BitCount, 1, 1, false, true, false},
	{ast.GetParam, 1, 1, false, true, false},
}
