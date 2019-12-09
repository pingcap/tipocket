package types


// SQLType enums for SQL types
type SQLType int

// SQLTypeDMLSelect
const (
	SQLTypeReloadSchema SQLType = iota
	SQLTypeDMLSelect
	SQLTypeDMLUpdate
	SQLTypeDMLInsert
	SQLTypeDMLDelete
	SQLTypeDDLCreate
	SQLTypeTxnBegin
	SQLTypeTxnCommit
	SQLTypeTxnRollback
	SQLTypeExec
	SQLTypeExit
	SQLTypeUnknown
)

// SQL struct
type SQL struct {
	SQLType SQLType
	SQLStmt string
}

func (t SQLType) String() string {
	switch t {
	case SQLTypeReloadSchema:
		return "SQLTypeReloadSchema"
	case SQLTypeDMLSelect:
		return "SQLTypeDMLSelect"
	case SQLTypeDMLUpdate:
		return "SQLTypeDMLUpdate"
	case SQLTypeDMLInsert:
		return "SQLTypeDMLInsert"
	case SQLTypeDMLDelete:
		return "SQLTypeDMLDelete"
	case SQLTypeDDLCreate:
		return "SQLTypeDDLCreate"
	case SQLTypeTxnBegin:
		return "SQLTypeTxnBegin"
	case SQLTypeTxnCommit:
		return "SQLTypeTxnCommit"
	case SQLTypeTxnRollback:
		return "SQLTypeTxnRollback"
	case SQLTypeExec:
		return "SQLTypeExec"
	case SQLTypeExit:
		return "SQLTypeExit"
	default:
		return "SQLTypeUnknown"
	}
}
