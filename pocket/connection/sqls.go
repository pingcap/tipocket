package connection

const (
	showDatabasesSQL  = "SHOW DATABASES"
	dropDatabaseSQL   = "DROP DATABASE IF EXISTS %s"
	createDatabaseSQL = "CREATE DATABASE %s"
	schemaSQL         = "SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE FROM information_schema.tables"
	tableSQL          = "DESC %s.%s"
	indexSQL          = "SHOW INDEX FROM %s.%s"
)
