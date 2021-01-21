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

var mathFunctions = []*functionClass{
	{ast.Abs, 1, 1, false, true, false},
	{ast.Acos, 1, 1, false, true, false},
	{ast.Asin, 1, 1, false, true, false},
	{ast.Atan, 1, 2, false, true, false},
	{ast.Atan2, 2, 2, false, true, false},
	{ast.Ceil, 1, 1, false, true, false},
	{ast.Ceiling, 1, 1, false, true, false},
	{ast.Conv, 3, 3, false, true, false},
	{ast.Cos, 1, 1, false, true, false},
	{ast.Cot, 1, 1, false, true, false},
	{ast.CRC32, 1, 1, false, true, false},
	{ast.Degrees, 1, 1, false, true, false},
	{ast.Exp, 1, 1, false, true, false},
	{ast.Floor, 1, 1, false, true, false},
	{ast.Ln, 1, 1, false, true, false},
	{ast.Log, 1, 2, false, true, false},
	{ast.Log2, 1, 1, false, true, false},
	{ast.Log10, 1, 1, false, true, false},
	{ast.PI, 0, 0, false, true, false},
	{ast.Pow, 2, 2, false, true, false},
	{ast.Power, 2, 2, false, true, false},
	{ast.Radians, 1, 1, false, true, false},
	{ast.Rand, 0, 1, false, true, true},
	{ast.Round, 1, 2, false, true, false},
	{ast.Sign, 1, 1, false, true, false},
	{ast.Sin, 1, 1, false, true, false},
	{ast.Sqrt, 1, 1, false, true, false},
	{ast.Tan, 1, 1, false, true, false},
	{ast.Truncate, 2, 2, false, true, false},
}
