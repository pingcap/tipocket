package model

import "github.com/pingcap/tipocket/pkg/scaffolds/file"

// Universe describes the entire state of file generation
type Universe struct {
	// Files contains the model of the files that are being scaffolded
	Files map[string]*file.File `json:"files,omitempty"`
}

// NewUniverse creates a new Universe
func NewUniverse() *Universe {
	return &Universe{}
}
