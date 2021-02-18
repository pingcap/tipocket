package template

import (
	"fmt"
	"github.com/pingcap/tipocket/pkg/scaffolds/file"
)

const (
	caseJsonnetMarker            = "// +tipocket:scaffold:case_decls"
	caseJsonnetInsertionTemplate = `  %[1]s(args={})::
    [
      '/bin/%[1]s',
    ],
`
)

// CaseJsonnetUpdater inserts a build rule for the new test case
type CaseJsonnetUpdater struct {
	file.InserterMixin
	CaseName string
}

// GetIfExistsAction ...
func (m *CaseJsonnetUpdater) GetIfExistsAction() file.IfExistsAction {
	return file.IfExistsActionOverwrite
}

// GetCodeFragments ...
func (m *CaseJsonnetUpdater) GetCodeFragments() map[file.Marker]file.CodeFragment {
	return map[file.Marker]file.CodeFragment{
		caseJsonnetMarker: {fmt.Sprintf(caseJsonnetInsertionTemplate, m.CaseName)},
	}
}
