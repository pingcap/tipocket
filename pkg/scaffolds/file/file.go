package file

// IfExistsAction determines what to do if the scaffold file already exists
type IfExistsAction int

const (
	// IfExistsActionError raises error if exist
	IfExistsActionError = iota
	// IfExistsActionOverwrite overwrite
	IfExistsActionOverwrite
)

// File stores persistent file
type File struct {
	// Path is the file to write
	Path string
	// Content is the generated output
	Content string

	// IfExistsAction determines what to do if the file exists
	IfExistsAction IfExistsAction
}
