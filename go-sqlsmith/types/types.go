package types 

// Database defines database database
type Database struct {
	Name string
	Tables map[string]*Table
}
