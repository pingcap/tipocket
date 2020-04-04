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
	LoadSchema(records [][5]string, indexes map[string][]string)
	// SetDB set operation database
	// the generated SQLs after this will be under this database
	SetDB(db string)
	// SetStable is a trigger for whether generate random or some database-basicinfo-dependent data
	// eg. SetStable(true) will disable both rand() and user() functions since they both make unstable result in different database
	SetStable(stable bool)
	// SetHint can control if hints would be generated or not
	SetHint(hint bool)
	// BeginWithOnlineTables to get online tables and begin transaction
	// if DMLOptions.OnlineTable is set to true
	// this function will return a table slice
	// and the tables beyond this slice will not be affected by DMLs during the generation
	BeginWithOnlineTables(opt *DMLOptions) []string
	// EndTransaction end transaction and remove online tables limitation
	EndTransaction() []string
	// SelectStmt generate select SQL
	SelectStmt(depth int) (string, string, error)
	// SelectForUpdateStmt generate select SQL with for update lock
	SelectForUpdateStmt(depth int) (string, string, error)
	// InsertStmt generate insert SQL
	InsertStmt(fn bool) (string, string, error)
	// UpdateStmt generate update SQL
	UpdateStmt() (string, string, error)
	// DeleteStmt generate delete SQL
	DeleteStmt() (string, string, error)
	// CreateTableStmt generate create table SQL
	CreateTableStmt() (string, string, error)
	// AlterTableStmt generate create table SQL, table slice for avoiding online DDL
	AlterTableStmt(opt *DDLOptions) (string, error)
	// CreateIndexStmt generate create index SQL, table slice for avoiding online DDL
	CreateIndexStmt(opt *DDLOptions) (string, error)
}

// DMLOptions for DML generation
type DMLOptions struct {
	// if OnlineTable is set to true
	// generator will rand some tables from the target database
	// and the following DML statements before transaction closed
	// should only affects these tables
	// else there will be no limit for affected tables
	// if you OnlineDDL field in DDLOptions is true, you may got the following error from TiDB
	// "ERROR 1105 (HY000): Information schema is changed. [try again later]"
	OnlineTable bool
}

// DDLOptions for DDL generation
type DDLOptions struct {
	// if OnlineDDL is set to false
	// DDL generation will look up tables other generators which are doing DMLs with the tables
	// the DDL generated will avoid modifing these tables
	// if OnlineDDL set to true
	// DDL generation will not avoid online tables
	OnlineDDL bool
	// if OnlineDDL is set to false
	// Tables contains all online tables which should not be modified with DDL
	// pocket will collect them from other generator instances
	Tables []string
}
