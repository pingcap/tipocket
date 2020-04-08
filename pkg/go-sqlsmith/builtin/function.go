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

import (
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/model"

	"github.com/pingcap/tipocket/pkg/go-sqlsmith/types"
	"github.com/pingcap/tipocket/pkg/go-sqlsmith/util"
)

// GenerateFuncCallExpr generate random builtin chain
func GenerateFuncCallExpr(table *types.Table, args int, stable bool) ast.ExprNode {
	if args == 0 && util.Rd(2) == 0 {
		return ast.NewValueExpr(util.GenerateRandDataItem(), "", "")
	}

	funcCallExpr := ast.FuncCallExpr{}

	fns := getValidArgsFunc(args, stable)
	fn := copyFunc(fns[util.Rd(len(fns))])
	funcCallExpr.FnName = model.NewCIStr(fn.name)
	for i := 0; i < args; i++ {
		r := util.Rd(100)
		if r > 80 {
			funcCallExpr.Args = append(funcCallExpr.Args, GenerateFuncCallExpr(table, 1+util.Rd(3), stable))
		}
		funcCallExpr.Args = append(funcCallExpr.Args, GenerateFuncCallExpr(table, 0, stable))
	}
	// if args != 0 {
	// 	log.Println(funcCallExpr)
	// 	for _, arg := range funcCallExpr.Args {
	// 		log.Println(arg)
	// 	}
	// }
	return &funcCallExpr
}

func getValidArgsFunc(args int, stable bool) []*functionClass {
	var fns []*functionClass
	for _, fn := range getFuncMap() {
		if fn.minArg > args {
			continue
		}
		if fn.stable != stable {
			continue
		}
		if fn.maxArg == -1 || fn.maxArg >= args {
			fns = append(fns, fn)
		}
	}
	return fns
}

func copyFunc(fn *functionClass) *functionClass {
	return &functionClass{
		name:     fn.name,
		minArg:   fn.minArg,
		maxArg:   fn.maxArg,
		constArg: fn.constArg,
		mysql:    fn.mysql,
	}
}
