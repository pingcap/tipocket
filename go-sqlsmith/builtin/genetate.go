// generate is for future usage
// which aim to make a builtin function combination or chain by a given type
// the returned builtin function combination can be multiple function call

package builtin

import (
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/model"
)

// GenerateTypeFuncCallExpr generate FuncCallExpr by given type
func GenerateTypeFuncCallExpr (t string) *ast.FuncCallExpr {
	switch t {
	case "timestamp":
		return GenerateTimestampFuncCallExpr()
	}
	return nil
}

// GenerateTimestampFuncCallExpr generate builtin func which will return date type
func GenerateTimestampFuncCallExpr () *ast.FuncCallExpr {
	return &ast.FuncCallExpr{
		FnName: model.NewCIStr("CURRENT_TIMESTAMP"),
	}
}
