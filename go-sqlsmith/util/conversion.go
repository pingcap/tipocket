package util


import "github.com/pingcap/parser/mysql"

// Type2Tp conver type string to tp byte
// TODO: complete conversion map
func Type2Tp(t string) byte {
	switch t {
	case "int":
		return mysql.TypeLong
	case "varchar":
		return mysql.TypeVarchar
	case "timestamp":
		return mysql.TypeTimestamp
	case "datetime":
		return mysql.TypeDatetime
	case "text":
		return mysql.TypeBlob
	}
	return mysql.TypeNull
}
