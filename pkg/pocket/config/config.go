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

	"github.com/pingcap/tipocket/pkg/pocket/pkg/types"
)

// Options struct
type Options struct {
	ClearDB       bool           `toml:"clear-db"`
	Stable        bool           `toml:"stable"`
	Reproduce     bool           `toml:"reproduce"`
	Concurrency   int            `toml:"concurrency"`
	InitTable     int            `toml:"init-table"`
	Path          string         `toml:"path"`
	Duration      types.Duration `toml:"duration"`
	CheckDuration types.Duration `toml:"check-duration"`
	OnlineDDL     bool           `toml:"online-ddl"`
	Serialize     bool           `toml:"serialize"`
	GeneralLog    bool           `toml:"general-log"`
}

// Generator Config
type Generator struct {
	SQLSmith SQLSmith
}

// Config struct
type Config struct {
	Mode      string    `toml:"mode"`
	DSN1      string    `toml:"dsn1"`
	DSN2      string    `toml:"dsn2"`
	Options   Options   `toml:"options"`
	Generator Generator `toml:"generator"`
}

var initConfig = Config{
	Mode: "single",
	DSN1: "root:@tcp(172.17.0.1:4000)/pocket",
	Options: Options{
		ClearDB:     false,
		Stable:      false,
		Reproduce:   false,
		Concurrency: 3,
		InitTable:   10,
		Path:        "./log",
		Duration: types.Duration{
			Duration: time.Hour,
		},
		CheckDuration: types.Duration{
			Duration: time.Minute,
		},
		OnlineDDL:  true,
		Serialize:  false,
		GeneralLog: false,
	},
	Generator: Generator{
		SQLSmith: SQLSmith{
			TxnBegin:           20,
			TxnCommit:          20,
			TxnRollback:        10,
			DDLCreateTable:     1,
			DDLAlterTable:      10,
			DDLCreateIndex:     10,
			DMLSelect:          10,
			DMLSelectForUpdate: 30,
			DMLDelete:          10,
			DMLUpdate:          120,
			DMLInsert:          120,
			Sleep:              10,
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
