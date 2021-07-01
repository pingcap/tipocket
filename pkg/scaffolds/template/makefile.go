package template

import (
	"fmt"

	"github.com/pingcap/tipocket/pkg/scaffolds/file"
)

const (
	makefileBuildMarker            = "# +tipocket:scaffold:makefile_build"
	makefileCmdMarker              = "# +tipocket:scaffold:makefile_build_cmd"
	makefileBuildInsertionTemplate = `    %s \
`
	makefileCmdInsertionTemplate = `%[1]s:
	cd testcase/%[1]s; make build; \
	cp bin/* ../../bin/

`
)

// MakefileUpdater inserts a build rule for the new test case
type MakefileUpdater struct {
	file.InserterMixin
	CaseName string
}

// GetIfExistsAction ...
func (m *MakefileUpdater) GetIfExistsAction() file.IfExistsAction {
	return file.IfExistsActionOverwrite
}

// GetCodeFragments ...
func (m *MakefileUpdater) GetCodeFragments() map[file.Marker]file.CodeFragment {
	return map[file.Marker]file.CodeFragment{
		makefileBuildMarker: {fmt.Sprintf(makefileBuildInsertionTemplate, m.CaseName)},
		makefileCmdMarker:   {fmt.Sprintf(makefileCmdInsertionTemplate, m.CaseName)},
	}
}
