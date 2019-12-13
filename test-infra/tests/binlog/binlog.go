// Copyright 2019 PingCAP, Inc.
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

package binlog

import (
	"github.com/onsi/ginkgo"
	"github.com/pingcap/tipocket/test-infra/tests/util"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = ginkgo.Describe("binlog", func() {

	f := framework.NewDefaultFramework("binlog")

	ginkgo.BeforeEach(func() {
		_ = f.Namespace.Name
		conf, err := framework.LoadConfig()
		framework.ExpectNoError(err, "Expected to load config")
		_ = util.NewE2eCli(conf)
	})

	ginkgo.It("should be deployed", func() {
		// TODO
	})
})
