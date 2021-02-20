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

package workflow

import "github.com/pingcap/tipocket/pkg/scaffolds/file"

// CaseJsonnetTemplate uses for client.go
type CaseJsonnetTemplate struct {
	file.TemplateMixin
	CaseName string
}

// GetIfExistsAction ...
func (c *CaseJsonnetTemplate) GetIfExistsAction() file.IfExistsAction {
	return file.IfExistsActionError
}

// Validate ...
func (c *CaseJsonnetTemplate) Validate() error {
	return c.TemplateMixin.Validate()
}

// SetTemplateDefaults ...
func (c *CaseJsonnetTemplate) SetTemplateDefaults() error {
	c.TemplateBody = clientTemplate
	return nil
}

const clientTemplate = `{
  _config+:: {
    case_name: '{{ .CaseName }}',
    image_name: 'hub.pingcap.net/qa/tipocket',
    args+: {
      // k8s configurations
      // 'storage-class': 'local-storage',
    },
    command: {},
  },
}
`
