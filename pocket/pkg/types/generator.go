package types

// Generator interface to unify the usage of sqlsmith and sqlspider
type Generator interface {
	// ReloadSchema function read raw scheme
	// For each record
	// record[0] dbname
	// record[1] table name
	// record[2] table type
	// record[3] column name
	// record[4] column type
	ReloadSchema([][5]string) error
	// SetDB set operation database
	// the generated SQLs after this will be under this database
	SetDB(db string)
	// SetStable is a trigger for whether generate random or some database-basicinfo-dependent data
	// eg. SetStable(true) will disable both rand() and user() functions since they both make unstable result in different database
	SetStable(stable bool)
	// SelectStmt generate select SQL
	SelectStmt() string
	// InsertStmt generate insert SQL
	InsertStmt() string
	// UpdateStmt generate update SQL
	UpdateStmt() string
	// CreateTableStmt generate create table SQL
	CreateTableStmt() string
}
