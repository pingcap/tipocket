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

package cmd

import (
	"github.com/pingcap/tipocket/pkg/scaffolds/file"
)

// Cmd uses for cmd/main.go
type Cmd struct {
	file.TemplateMixin
	CaseName string
}

// GetIfExistsAction ...
func (c *Cmd) GetIfExistsAction() file.IfExistsAction {
	return file.IfExistsActionError
}

// Validate ...
func (c *Cmd) Validate() error {
	return c.TemplateMixin.Validate()
}

// SetTemplateDefaults ...
func (c *Cmd) SetTemplateDefaults() error {
	c.TemplateBody = cmdTemplate
	return nil
}

const cmdTemplate = `// Copyright 2021 PingCAP, Inc.
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

package main

import (
	"context"
	"flag"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/cmd/util"
	logs "github.com/pingcap/tipocket/logsearch/pkg/logs"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"

	testcase "github.com/pingcap/tipocket/testcase/{{ .CaseName }}"
)

func main() {
	flag.Parse()
	cfg := control.Config{
		Mode:        control.ModeStandard,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
	}
	c := fixture.Context
	suit := util.Suit{
		Config:        &cfg,
		Provider:      cluster.NewDefaultClusterProvider(),
		ClientCreator: testcase.CaseCreator{},
		NemesisGens:   util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs:   test_infra.NewDefaultCluster(c.Namespace, c.Namespace, c.TiDBClusterConfig),
		LogsClient:    logs.NewDiagnosticLogClient(),
	}
	suit.Run(context.Background())
}
`
