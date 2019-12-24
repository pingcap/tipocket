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

package core

import (
	"context"
	"fmt"
)

// DB allows Chaos to set up and tear down database.
type DB interface {
	// SetUp initializes the database.
	SetUp(ctx context.Context, nodes []string, node string) error
	// TearDown tears down the database.
	TearDown(ctx context.Context, nodes []string, node string) error
	// Start starts the database
	Start(ctx context.Context, node string) error
	// Stop stops the database
	Stop(ctx context.Context, node string) error
	// Kill kills the database
	Kill(ctx context.Context, node string) error
	// IsRunning checks whether the database is running or not
	IsRunning(ctx context.Context, node string) bool
	// Name returns the unique name for the database
	Name() string
}

// NoopDB is a DB but does nothing
type NoopDB struct {
}

// SetUp initializes the database.
func (NoopDB) SetUp(ctx context.Context, nodes []string, node string) error {
	return nil
}

// TearDown tears down the datase.
func (NoopDB) TearDown(ctx context.Context, nodes []string, node string) error {
	return nil
}

// Start starts the database
func (NoopDB) Start(ctx context.Context, node string) error {
	return nil
}

// Stop stops the database
func (NoopDB) Stop(ctx context.Context, node string) error {
	return nil
}

// Kill kills the database
func (NoopDB) Kill(ctx context.Context, node string) error {
	return nil
}

// IsRunning checks whether the database is running or not
func (NoopDB) IsRunning(ctx context.Context, node string) bool {
	return true
}

// Name returns the unique name for the database
func (NoopDB) Name() string {
	return "noop"
}

var dbs = map[string]DB{}

// RegisterDB registers db. Not thread-safe
func RegisterDB(db DB) {
	name := db.Name()
	_, ok := dbs[name]
	if ok {
		panic(fmt.Sprintf("db %s is already registered", name))
	}

	dbs[name] = db
}

// GetDB gets the registered db.
func GetDB(name string) DB {
	return dbs[name]
}

func init() {
	RegisterDB(NoopDB{})
}
