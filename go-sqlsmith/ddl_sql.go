package sqlsmith


// CreateTableStmt create table
func (s *SQLSmith) CreateTableStmt() (string, error) {
	tree := s.createTableStmt()
	return s.Walk(tree)
}

// AlterTableStmt alter table
func (s *SQLSmith) AlterTableStmt() (string, error) {
	tree := s.alterTableStmt()
	return s.Walk(tree)
}

// CreateIndexStmt create index
func (s *SQLSmith) CreateIndexStmt() (string, error) {
	tree := s.createIndexStmt()
	return s.Walk(tree)
}
