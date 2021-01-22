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

package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatTimeStrAsLog(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	assert.Equal(t, "2019/08/10 11:45:14.000 +08:00", FormatTimeStrAsLog(time.Unix(1565408714, 0).In(loc)))
}
