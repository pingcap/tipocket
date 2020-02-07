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

var stringFunctions = []*functionClass{
	{ast.ASCII, 1, 1, false, true, false},
	{ast.Bin, 1, 1, false, true, false},
	{ast.Concat, 1, -1, false, true, false},
	{ast.ConcatWS, 2, -1, false, true, false},
	{ast.Convert, 2, 2, false, true, false},
	{ast.Elt, 2, -1, false, true, false},
	{ast.ExportSet, 3, 5, false, true, false},
	{ast.Field, 2, -1, false, true, false},
	{ast.Format, 2, 3, false, true, false},
	{ast.FromBase64, 1, 1, false, true, false},
	{ast.InsertFunc, 4, 4, false, true, false},
	{ast.Instr, 2, 2, false, true, false},
	{ast.Lcase, 1, 1, false, true, false},
	{ast.Left, 2, 2, false, true, false},
	{ast.Right, 2, 2, false, true, false},
	{ast.Length, 1, 1, false, true, false},
	// {ast.LoadFile, 1, 1, false, true, false},
	{ast.Locate, 2, 3, false, true, false},
	{ast.Lower, 1, 1, false, true, false},
	{ast.Lpad, 3, 3, false, true, false},
	{ast.LTrim, 1, 1, false, true, false},
	{ast.Mid, 3, 3, false, true, false},
	{ast.MakeSet, 2, -1, false, true, false},
	{ast.Oct, 1, 1, false, true, false},
	{ast.OctetLength, 1, 1, false, true, false},
	{ast.Ord, 1, 1, false, true, false},
	{ast.Position, 2, 2, false, true, false},
	{ast.Quote, 1, 1, false, true, false},
	{ast.Repeat, 2, 2, false, true, false},
	{ast.Replace, 3, 3, false, true, false},
	{ast.Reverse, 1, 1, false, true, false},
	{ast.RTrim, 1, 1, false, true, false},
	{ast.Space, 1, 1, false, true, false},
	{ast.Strcmp, 2, 2, false, true, false},
	{ast.Substring, 2, 3, false, true, false},
	{ast.Substr, 2, 3, false, true, false},
	{ast.SubstringIndex, 3, 3, false, true, false},
	{ast.ToBase64, 1, 1, false, true, false},
	// not support fully trim, since the trim expression is special
	// SELECT TRIM('miyuri' FROM 'miyuri-jinja-no-miyuri');
	// {ast.Trim, 1, 3, false, true, false},
	// {ast.Trim, 1, 1, false, true},
	{ast.Upper, 1, 1, false, true, false},
	{ast.Ucase, 1, 1, false, true, false},
	{ast.Hex, 1, 1, false, true, false},
	{ast.Unhex, 1, 1, false, true, false},
	{ast.Rpad, 3, 3, false, true, false},
	{ast.BitLength, 1, 1, false, true, false},
	// {ast.CharFunc, 2, -1, false, true, false},
	{ast.CharLength, 1, 1, false, true, false},
	{ast.CharacterLength, 1, 1, false, true, false},
	{ast.FindInSet, 2, 2, false, true, false},
}
