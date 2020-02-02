package config

// SQLSmith for sqlsmith generator configuration only
type SQLSmith struct {
	TxnBegin           int `toml:"txn-begin"`
	TxnCommit          int `toml:"txn-commit"`
	TxnRollback        int `toml:"txn-rollback"`
	DDLCreateTable     int `toml:"ddl-create-table"`
	DDLAlterTable      int `toml:"ddl-alter-table"`
	DDLCreateIndex     int `toml:"ddl-create-index"`
	DMLSelect          int `toml:"dml-select"`
	DMLSelectForUpdate int `toml:"dml-select-for-update"`
	DMLDelete          int `toml:"dml-delete"`
	DMLUpdate          int `toml:"dml-update"`
	DMLInsert          int `toml:"dml-insert"`
	Sleep              int `toml:"sleep"`
}
