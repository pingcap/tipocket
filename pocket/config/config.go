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
	"time"

	"github.com/BurntSushi/toml"
	"github.com/juju/errors"
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

// Options struct
type Options struct {
	ClearDB     bool           `toml:"clear-db"`
	Stable      bool           `toml:"stable"`
	Reproduce   bool           `toml:"reproduce"`
	Concurrency int            `toml:"concurrency"`
	Path        string         `toml:"path"`
	Duration    types.Duration `toml:"duration"`
}

// Config struct
type Config struct {
	Mode    string  `toml:"mode"`
	Dsn1    string  `toml:"dsn1"`
	Dsn2    string  `toml:"dsn2"`
	Options Options `toml:"options"`
}

var initConfig = Config{
	Mode: "single",
	Dsn1: "127.0.0.1:4000",
	Options: Options{
		ClearDB: false,
		Stable: false,
		Reproduce: false,
		Concurrency: 3,
		Path: "./log",
		Duration: types.Duration{
			Duration: time.Hour,
		},
	},
}

// Init get default Config
func Init() *Config {
	return initConfig.Copy()
}

// Load config from file 
func (c *Config) Load(path string) error {
	_, err := toml.DecodeFile(path, c)
	return errors.Trace(err)
}

// Copy Config struct
func (c *Config) Copy() *Config {
	cp := *c
	return &cp
}
