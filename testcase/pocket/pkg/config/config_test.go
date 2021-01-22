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

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	config := Init()
	config.Options.Duration.Duration = 24 * time.Hour

	assert.Empty(t, config.Load("./config.example.toml"))
	assert.Equal(t, config.Mode, "abtest")
	assert.Equal(t, config.DSN1, "root:@tcp(172.17.0.1:33306)/pocket")
	assert.Equal(t, config.DSN2, "root:@tcp(172.17.0.1:4000)/pocket")
	assert.Equal(t, config.Options.Duration.Duration, time.Hour)
	assert.Equal(t, config.Options.CheckDuration.Duration, time.Minute)
	assert.Equal(t, config.Options.Path, "/var/log/pocket")
}
