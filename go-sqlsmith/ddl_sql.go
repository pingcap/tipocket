package sqlsmith


// CreateTableStmt create table
func (s *SQLSmith) CreateTableStmt() (string, error) {
	tree := s.createTableStmt()
	return s.Walk(tree)
}
