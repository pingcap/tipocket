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

// generate is for future usage
// which aim to make a builtin function combination or chain by a given type
// the returned builtin function combination can be multiple function call

package builtin

import (
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/model"
)

// GenerateTypeFuncCallExpr generate FuncCallExpr by given type
func GenerateTypeFuncCallExpr(t string) *ast.FuncCallExpr {
	switch t {
	case "timestamp":
		return GenerateTimestampFuncCallExpr()
	}
	return nil
}

// GenerateTimestampFuncCallExpr generate builtin func which will return date type
func GenerateTimestampFuncCallExpr() *ast.FuncCallExpr {
	return &ast.FuncCallExpr{
		FnName: model.NewCIStr("CURRENT_TIMESTAMP"),
	}
}
