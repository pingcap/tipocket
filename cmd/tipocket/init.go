package main

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/pingcap/tipocket/pkg/scaffolds"
	"github.com/pingcap/tipocket/pkg/scaffolds/file"
	"github.com/pingcap/tipocket/pkg/scaffolds/model"
	"github.com/pingcap/tipocket/pkg/scaffolds/template"
	"github.com/pingcap/tipocket/pkg/scaffolds/template/testcase"
	testcasecmd "github.com/pingcap/tipocket/pkg/scaffolds/template/testcase/cmd"
	"github.com/pingcap/tipocket/pkg/scaffolds/template/workflow"
)

var (
	caseNameFlag string
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialize a new test case",
		Example: "",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			r, _ := regexp.Compile("^[a-z][a-z_]*$")
			if !r.MatchString(caseNameFlag) {
				return fmt.Errorf("case-name must in the form of [a-z][a-z_]*")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			scaffolder := scaffolds.NewScaffold()
			universe := model.NewUniverse()
			err := scaffolder.Execute(universe, &testcase.Makefile{
				TemplateMixin: file.TemplateMixin{Path: filepath.Join("testcase", caseNameFlag, "Makefile")},
				CaseName:      caseNameFlag,
			}, &testcase.Client{
				TemplateMixin: file.TemplateMixin{Path: filepath.Join("testcase", caseNameFlag, "client.go")},
				CaseName:      caseNameFlag,
			}, &testcasecmd.Cmd{
				TemplateMixin: file.TemplateMixin{Path: filepath.Join("testcase", caseNameFlag, "cmd", "main.go")},
				CaseName:      caseNameFlag,
			}, &testcase.GoModule{
				TemplateMixin: file.TemplateMixin{Path: filepath.Join("testcase", caseNameFlag, "go.mod")},
				CaseName:      caseNameFlag,
			}, &testcase.Revive{
				TemplateMixin: file.TemplateMixin{Path: filepath.Join("testcase", caseNameFlag, "revive.toml")},
				CaseName:      caseNameFlag,
			}, &template.MakefileUpdater{
				InserterMixin: file.InserterMixin{Path: "Makefile"},
				CaseName:      caseNameFlag,
			}, &template.CaseJsonnetUpdater{
				InserterMixin: file.InserterMixin{Path: filepath.Join("run", "lib", "case.jsonnet")},
				CaseName:      caseNameFlag,
			}, &workflow.CaseJsonnetTemplate{
				TemplateMixin: file.TemplateMixin{Path: filepath.Join("run", "workflow", fmt.Sprintf("%s.jsonnet", caseNameFlag))},
				CaseName:      caseNameFlag,
			})
			if err != nil {
				return err
			}
			fmt.Printf("create a new case `%[1]s`: testcase/%[1]s\n", caseNameFlag)
			return nil
		},
	}
	cmd.Flags().StringVarP(&caseNameFlag, "case-name", "c", "", "test case name")
	cmd.MarkFlagRequired("case-name")
	return cmd
}
