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

var miscellaneousFunctions = []*functionClass{
	{ast.AnyValue, 1, 1, false, true, false},
	{ast.InetAton, 1, 1, false, true, false},
	{ast.InetNtoa, 1, 1, false, true, false},
	{ast.Inet6Aton, 1, 1, false, true, false},
	{ast.Inet6Ntoa, 1, 1, false, true, false},
	// {ast.IsFreeLock, 1, 1, false, true, false},
	{ast.IsIPv4, 1, 1, false, true, false},
	{ast.IsIPv4Compat, 1, 1, false, true, false},
	{ast.IsIPv4Mapped, 1, 1, false, true, false},
	{ast.IsIPv6, 1, 1, false, true, false},
	// {ast.IsUsedLock, 1, 1, false, true, false},
	// {ast.MasterPosWait, 2, 4, false, true, false},
	{ast.NameConst, 2, 2, false, true, false},
	// {ast.ReleaseAllLocks, 0, 0, false, true, false},
	{ast.UUID, 0, 0, false, true, true},
	// {ast.UUIDShort, 0, 0, false, true, false},
}
