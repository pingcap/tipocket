// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package file

// Builder defines the basic methods that any file builder must implement
type Builder interface {
	// GetPath returns the path to the file location
	GetPath() string
	Validate() error
	GetIfExistsAction() IfExistsAction
}

// Template is the file builder based on a template file
type Template interface {
	Builder
	GetBody() string
	SetTemplateDefaults() error
}

// Inserter is a file builder that inserts code fragments in marked positions
type Inserter interface {
	Builder
	// GetCodeFragments returns a map that binds markers to code fragments
	GetCodeFragments() map[Marker]CodeFragment
}
