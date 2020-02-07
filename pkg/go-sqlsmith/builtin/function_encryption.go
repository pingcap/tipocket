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

// compress and uncompress function make diff
var encryptionFunctions = []*functionClass{
	{ast.AesDecrypt, 2, 3, false, true, false},
	{ast.AesEncrypt, 2, 3, false, true, false},
	// {ast.Compress, 1, 1, false, true, false},
	// removed in MySQL 8.0.3
	// {ast.Decode, 2, 2, false, true, false},
	{ast.DesDecrypt, 1, 2, false, true, false},
	{ast.DesEncrypt, 1, 2, false, true, false},
	// {ast.Encode, 2, 2, false, true, false},
	{ast.Encrypt, 1, 2, false, true, false},
	{ast.MD5, 1, 1, false, true, false},
	{ast.OldPassword, 1, 1, false, true, false},
	// {ast.PasswordFunc, 1, 1, false, true, false},
	{ast.RandomBytes, 1, 1, false, true, false},
	{ast.SHA1, 1, 1, false, true, false},
	{ast.SHA, 1, 1, false, true, false},
	{ast.SHA2, 2, 2, false, true, false},
	// {ast.Uncompress, 1, 1, false, true, false},
	// {ast.UncompressedLength, 1, 1, false, true, false},
	// {ast.ValidatePasswordStrength, 1, 1, false, true, false},
}
