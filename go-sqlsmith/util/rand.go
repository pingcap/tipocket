package util


import (
	"fmt"
	"math/rand"
	"time"
	"github.com/pingcap/parser/ast"
)

var (
	seed = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func Rd (n int) int {
	return seed.Intn(n)
}

func RdRange (n, m int) int {
	if n == m {
		return n
	}
	if m < n {
		n, m = m, n
	}
	return n + seed.Intn(m - n)
}

func RdFloat64() float64 {
	return seed.Float64()
}

func RdDate() time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2100, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min

	sec := seed.Int63n(delta) + min
	return time.Unix(sec, 0)
}

func RdString (length int) string {
	res := ""
	for i := 0; i < length; i++ {
		charCode := RdRange(33, 127)
		// char '\' and '"' should be escaped
		if charCode == 92 || charCode == 34 {
			charCode++
			// res = fmt.Sprintf("%s%s", res, "\\")
		}
		res = fmt.Sprintf("%s%s", res, string(rune(charCode)))
	}
	return res
}

func RdStringChar (length int) string {
	res := ""
	for i := 0; i < length; i++ {
		charCode := RdRange(97, 123)
		res = fmt.Sprintf("%s%s", res, string(rune(charCode)))
	}
	return res
}

func RdType () string {
	switch Rd(6) {
	case 0:
		return "varchar"
	case 1:
		return "text"
	case 2:
		return "timestamp"
	case 3:
		return "datetime"
	}
	return "int"
}

func RdDataLen(columnType string) int {
	switch columnType {
	case "int":
		return RdRange(8, 20)
	case "varchar":
		return RdRange(255, 2047)
	case "float":
		return RdRange(16, 64)
	case "timestamp":
		return -1
	case "datetime":
		return -1
	case "text":
		return -1
	}
	return 10
}

func RdColumnOptions(t string) []ast.ColumnOptionType {
	switch t {
	case "timestamp":
		return RdDateColumnOptions()
	}

	return []ast.ColumnOptionType{}
}

func RdDateColumnOptions() (options []ast.ColumnOptionType) {
	options = append(options, ast.ColumnOptionNotNull)
	if Rd(2) == 0 {
		options = append(options, ast.ColumnOptionDefaultValue)
	}
	return
}

func RdCharset() string {
	switch Rd(4) {
	default:
		return "utf8"
	}
}
