package types

import "github.com/pingcap/parser/ast"

// Column defines database column
type Column struct {
	DB string
	Table string
	OriginTable string
	Column string
	OriginColumn string
	DataType string
	DataLen int
	Func bool
	NewFunc bool
	Options []ast.ColumnOptionType
}

// Clone makes a replica of column
func (c *Column) Clone() *Column {
	column := *c
	return &column
}

// AddOption add option for column
func (c *Column) AddOption(opt ast.ColumnOptionType) {
	for _, option := range c.Options {
		if option == opt {
			return
		}
	}
	c.Options = append(c.Options, opt)
}

// HasOption return is has the given option
func (c *Column) HasOption(opt ast.ColumnOptionType) bool {
	for _, option := range c.Options {
		if option == opt {
			return true
		}
	}
	return false
}
