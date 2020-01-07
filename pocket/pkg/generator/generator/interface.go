package generator

// Generator interface to unify the usage of sqlsmith and sqlspider
type Generator interface {
	// ReloadSchema function read raw scheme
	// For each record
	// record[0] dbname
	// record[1] table name
	// record[2] table type
	// record[3] column name
	// record[4] column type
	// indexes map table name to index string slice
	LoadSchema (records [][5]string, indexes map[string][]string)
	// SetDB set operation database
	// the generated SQLs after this will be under this database
	SetDB(db string)
	// SetStable is a trigger for whether generate random or some database-basicinfo-dependent data
	// eg. SetStable(true) will disable both rand() and user() functions since they both make unstable result in different database
	SetStable(stable bool)
	// BeginWithOnlineTables to get online tables and begin transaction
	BeginWithOnlineTables(opt *DMLOptions) []string
	// EndTransaction end transaction and remove online tables limitation
	EndTransaction() []string
	// SelectStmt generate select SQL
	SelectStmt(depth int) (string, error)
	// InsertStmt generate insert SQL
	InsertStmt(fn bool) (string, string, error)
	// UpdateStmt generate update SQL
	UpdateStmt() (string, string, error)
	// CreateTableStmt generate create table SQL
	CreateTableStmt() (string, error)
	// AlterTableStmt generate create table SQL, table slice for avoiding online DDL
	AlterTableStmt(opt *DDLOptions) (string, error)
	// CreateIndexStmt generate create index SQL, table slice for avoiding online DDL
	CreateIndexStmt(opt *DDLOptions) (string, error)
}

// DMLOptions for DML generation
type DMLOptions struct {
	OnlineTable bool
}

// DDLOptions for DDL generation
type DDLOptions struct {
	OnlineDDL bool
	Tables    []string
}
