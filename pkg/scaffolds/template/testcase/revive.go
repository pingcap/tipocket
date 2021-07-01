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

package testcase

import (
	"github.com/pingcap/tipocket/pkg/scaffolds/file"
)

// Revive ...
type Revive struct {
	file.TemplateMixin
	// CaseName is test case name
	CaseName string
}

// GetIfExistsAction ...
func (r *Revive) GetIfExistsAction() file.IfExistsAction {
	return file.IfExistsActionError
}

// Validate ...
func (r *Revive) Validate() error {
	return r.TemplateMixin.Validate()
}

// SetTemplateDefaults ...
func (r *Revive) SetTemplateDefaults() error {
	r.TemplateBody = reviveTemplate
	return nil
}

const reviveTemplate = `ignoreGeneratedHeader = false
severity = "error"
confidence = 0.8
errorCode = -1
warningCode = -1

[rule.blank-imports]
[rule.context-as-argument]
[rule.dot-imports]
[rule.error-return]
[rule.error-strings]
[rule.error-naming]
[rule.exported]
#[rule.if-return]
[rule.var-naming]
[rule.package-comments]
[rule.range]
[rule.receiver-naming]
[rule.indent-error-flow]
#[rule.superfluous-else]
[rule.modifies-parameter]

# This can be checked by other tools like megacheck
[rule.unreachable-code]


# Currently this makes too much noise, but should add it in
# and perhaps ignore it in a few files
#[rule.confusing-naming]
#  severity = "warning"
#[rule.confusing-results]
#  severity = "warning"
#[rule.unused-parameter]
#  severity = "warning"
#[rule.deep-exit]
#  severity = "warning"
#[rule.flag-parameter]
#  severity = "warning"



# Adding these will slow down the linter
# They are already provided by megacheck
# [rule.unexported-return]
# [rule.time-naming]
# [rule.errorf]

# Adding these will slow down the linter
# Not sure if they are already provided by megacheck
# [rule.var-declaration]
# [rule.context-keys-type]
`
