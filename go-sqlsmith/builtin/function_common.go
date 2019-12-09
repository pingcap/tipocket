package builtin


import "github.com/pingcap/parser/ast"

var commonFunctions = []*functionClass{
	{ast.Coalesce, 1, -1, false, true, false},
	{ast.IsNull, 1, 1, false, true, false},
	{ast.Greatest, 2, -1, false, true, false},
	{ast.Least, 2, -1, false, true, false},
	{ast.Interval, 2, -1, false, true, false},
}
