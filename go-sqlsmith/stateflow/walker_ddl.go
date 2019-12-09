package stateflow

import (
	"fmt"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/model"
	parserTypes "github.com/pingcap/parser/types"
	tidbTypes "github.com/pingcap/tidb/types"
	"github.com/pingcap/tipocket/go-sqlsmith/builtin"
	"github.com/pingcap/tipocket/go-sqlsmith/types"
	"github.com/pingcap/tipocket/go-sqlsmith/util"
	"github.com/pingcap/tidb/types/parser_driver"
)

const (
	timeParseFormat = "2006-01-02 15:04:05"
)

var (
	intPartition = []int64{1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11}
	datetimePartition = []string{"1980-01-01 00:00:00", "1990-01-01 00:00:00", "2000-01-01 00:00:00", "2010-01-01 00:00:00", "2020-01-01 00:00:00"}
	timestampPartition = []string{"1980-01-01 00:00:00", "1990-01-01 00:00:00", "2000-01-01 00:00:00", "2010-01-01 00:00:00", "2020-01-01 00:00:00"}
)

func (s *StateFlow) walkCreateTableStmt(node *ast.CreateTableStmt) *types.Table {
	table := s.randNewTable()
	for _, column := range table.Columns {
		node.Cols = append(node.Cols, &ast.ColumnDef{
			Name: &ast.ColumnName{
				Name: model.NewCIStr(column.Column),
			},
			Tp: s.makeFieldType(column.DataType, column.DataLen),
			Options: s.makeColumnOptions(column, column.Options),
		})
		if column.HasOption(ast.ColumnOptionAutoIncrement) {
			s.makeConstraintPrimaryKey(node, column)
		}
	}
	node.Table.Name = model.NewCIStr(table.Table)
	s.walkTableOption(node)
	if column := table.RandColumn(); node.Partition != nil && column != nil {
		s.walkPartition(node.Partition, column)
		if column.Column != "id" {
			s.makeConstraintPrimaryKey(node, column)
		}
	}
	return nil
}

func (s *StateFlow) makeFieldType(t string, l int) *parserTypes.FieldType {
	fieldType := parserTypes.NewFieldType(util.Type2Tp(t))
	fieldType.Flen = l
	return fieldType
}

func (s *StateFlow) makeColumnOptions(column *types.Column, options []ast.ColumnOptionType) (columnOptions []*ast.ColumnOption) {
	for _, opt := range options {
		columnOptions = append(columnOptions, s.makeColumnOption(column, opt))
	}
	return
}

func (s *StateFlow) makeColumnOption(column *types.Column, option ast.ColumnOptionType) *ast.ColumnOption {
	columnOption := ast.ColumnOption{
		Tp: option,
	}
	if option == ast.ColumnOptionDefaultValue {
		columnOption.Expr = builtin.GenerateTypeFuncCallExpr(column.DataType)
	}
	return &columnOption
}

// makeConstraintPromaryKey is for convenience
func (s *StateFlow) makeConstraintPrimaryKey(node *ast.CreateTableStmt, column *types.Column) {
	for _, constraint := range node.Constraints {
		if constraint.Tp == ast.ConstraintPrimaryKey {
			constraint.Keys = append(constraint.Keys, &ast.IndexColName{
				Column: &ast.ColumnName{
					Name: model.NewCIStr(column.Column),
				},
			})
			return
		}
	}
	node.Constraints = append(node.Constraints, &ast.Constraint{
		Tp: ast.ConstraintPrimaryKey,
		Keys: []*ast.IndexColName{
			{
				Column: &ast.ColumnName{
					Name: model.NewCIStr(column.Column),
				},
			},
		},
	})
}

func (s *StateFlow) walkTableOption(node *ast.CreateTableStmt) {
	node.Options = append(node.Options, &ast.TableOption{
		Tp: ast.TableOptionEngine,
		StrValue: "InnoDB",
	})
	node.Options = append(node.Options, &ast.TableOption{
		Tp: ast.TableOptionCharset,
		StrValue: util.RdCharset(),
	})
}

func (s *StateFlow) walkPartition(node *ast.PartitionOptions, column *types.Column) {
	// set partition Tp
	node.Tp = model.PartitionTypeRange

	// set to int func
	var funcCallNode = new(ast.FuncCallExpr)
	switch column.DataType {
	case "timestamp":
		funcCallNode.FnName = model.NewCIStr("UNIX_TIMESTAMP")
	case "datetime":
		funcCallNode.FnName = model.NewCIStr("TO_DAYS")
	case "varchar", "text":
		funcCallNode.FnName = model.NewCIStr("ASCII")
	}

	// partition by column
	partitionByFuncCall := 	funcCallNode
	if funcCallNode.FnName.String() == "" {
		node.Expr = &ast.ColumnNameExpr{
			Name: &ast.ColumnName{
				Name: model.NewCIStr(column.Column),
			},
		}
	} else {
		partitionByFuncCall.Args = []ast.ExprNode{
			&ast.ColumnNameExpr{
				Name: &ast.ColumnName{
					Name: model.NewCIStr(column.Column),
				},
			},
		}
		node.Expr = partitionByFuncCall
	}

	// set partition definitions
	s.walkPartitionDefinitions(&node.Definitions, column)
}

func (s *StateFlow) walkPartitionDefinitions(definitions *[]*ast.PartitionDefinition, column *types.Column) {
	switch column.DataType {
	case "int":
		s.walkPartitionDefinitionsInt(definitions)
	case "varchar", "text":
		s.walkPartitionDefinitionsString(definitions)
	case "datetime":
		s.walkPartitionDefinitionsDatetime(definitions)
	case "timestamp":
		s.walkPartitionDefinitionsTimestamp(definitions)
	}

	*definitions = append(*definitions, &ast.PartitionDefinition{
		Name: model.NewCIStr("pn"),
		Clause: &ast.PartitionDefinitionClauseLessThan{
			Exprs:[]ast.ExprNode{
				&ast.MaxValueExpr{},
			},
		},
	})
}

func (s *StateFlow) walkPartitionDefinitionsInt(definitions *[]*ast.PartitionDefinition) {
	for i := 0; i < len(intPartition); i += util.RdRange(1, 3) {
		val := driver.ValueExpr{}
		val.SetInt64(intPartition[i])
		*definitions = append(*definitions, &ast.PartitionDefinition{
				Name: model.NewCIStr(fmt.Sprintf("p%d", i)),
				Clause: &ast.PartitionDefinitionClauseLessThan{
					Exprs:[]ast.ExprNode{
						&val,
					},
				},
			})
	}
}

func (s *StateFlow) walkPartitionDefinitionsString(definitions *[]*ast.PartitionDefinition) {
	for i := 0; i < 256; i += util.RdRange(1, 10) {
		val := driver.ValueExpr{}
		val.SetInt64(int64(i))
		*definitions = append(*definitions, &ast.PartitionDefinition{
				Name: model.NewCIStr(fmt.Sprintf("p%d", i)),
				Clause: &ast.PartitionDefinitionClauseLessThan{
					Exprs:[]ast.ExprNode{
						&ast.FuncCallExpr{
							FnName: model.NewCIStr("ASCII"),
							Args: []ast.ExprNode{&val},
						},
					},
				},
			})
	}
}

func (s *StateFlow) walkPartitionDefinitionsDatetime(definitions *[]*ast.PartitionDefinition) {
	for i := 0; i < len(datetimePartition); i += util.RdRange(1, 3) {
		val := driver.ValueExpr{}
		val.SetMysqlTime(tidbTypes.Time{
			Time: tidbTypes.FromGoTime(util.TimeMustParse(timeParseFormat, datetimePartition[i])),
		})
		*definitions = append(*definitions, &ast.PartitionDefinition{
				Name: model.NewCIStr(fmt.Sprintf("p%d", i)),
				Clause: &ast.PartitionDefinitionClauseLessThan{
					Exprs:[]ast.ExprNode{
						&ast.FuncCallExpr{
							FnName: model.NewCIStr("TO_DAYS"),
							Args: []ast.ExprNode{&val},
						},
					},
				},
			})
	}
}

func (s *StateFlow) walkPartitionDefinitionsTimestamp(definitions *[]*ast.PartitionDefinition) {
	for i := 0; i < len(timestampPartition); i += util.RdRange(1, 3) {
		val := driver.ValueExpr{}
		val.SetMysqlTime(tidbTypes.Time{
			Time: tidbTypes.FromGoTime(util.TimeMustParse(timeParseFormat, timestampPartition[i])),
		})
		*definitions = append(*definitions, &ast.PartitionDefinition{
				Name: model.NewCIStr(fmt.Sprintf("p%d", i)),
				Clause: &ast.PartitionDefinitionClauseLessThan{
					Exprs:[]ast.ExprNode{
						&ast.FuncCallExpr{
							FnName: model.NewCIStr("TO_DAYS"),
							Args: []ast.ExprNode{&val},
						},
					},
				},
			})
	}
}
