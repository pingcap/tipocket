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
	caseNameFlag        string
	individualBuildFlag bool
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialize a new test case",
		Example: "",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			r, _ := regexp.Compile("^[a-z][a-z-]*[a-z0-9]+$")
			if !r.MatchString(caseNameFlag) {
				return fmt.Errorf("case-name must in the form of [a-z][a-z-]*[a-z0-9]+")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			scaffolder := scaffolds.NewScaffold()
			universe := model.NewUniverse()
			fileBuilders := []file.Builder{&testcase.Makefile{
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
				InserterMixin:   file.InserterMixin{Path: "Makefile"},
				CaseName:        caseNameFlag,
				IndividualBuild: individualBuildFlag,
			}, &template.CaseJsonnetUpdater{
				InserterMixin: file.InserterMixin{Path: filepath.Join("run", "lib", "case.libsonnet")},
				CaseName:      caseNameFlag,
			}, &workflow.CaseJsonnetTemplate{
				TemplateMixin:   file.TemplateMixin{Path: filepath.Join("run", "workflow", fmt.Sprintf("%s.jsonnet", caseNameFlag))},
				CaseName:        caseNameFlag,
				IndividualBuild: individualBuildFlag,
			}}
			if individualBuildFlag {
				fileBuilders = append(fileBuilders, &testcase.Dockerfile{
					TemplateMixin: file.TemplateMixin{Path: filepath.Join("testcase", caseNameFlag, "Dockerfile")},
					CaseName:      caseNameFlag,
				})
			}
			if err := scaffolder.Execute(universe, fileBuilders...); err != nil {
				return err
			}
			fmt.Printf("create a new case `%[1]s`: testcase/%[1]s\n", caseNameFlag)
			return nil
		},
	}
	cmd.Flags().StringVarP(&caseNameFlag, "case-name", "c", "", "test case name")
	cmd.Flags().BoolVarP(&individualBuildFlag, "individual-build", "i", false, "individual build from root image")
	cmd.MarkFlagRequired("case-name")
	return cmd
}
