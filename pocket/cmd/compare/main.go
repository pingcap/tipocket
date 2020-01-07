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

package main

import (
	"flag"
	"fmt"
	"regexp"

	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/pocket/connection"
	"github.com/pingcap/tipocket/pocket/executor"
)

var (
	matchDB = regexp.MustCompile("([0-9a-zA-Z]+)$")
	selectStmt = "SELECT * FROM %s"
)

var (
	nmDsn1 = "dsn1"
	nmDsn2 = "dsn2"
)

var (
	dsn1 = flag.String(nmDsn1, "", "dsn1")
	dsn2 = flag.String(nmDsn2, "", "dsn2")
)

func main() {
	flag.Parse()
	if *dsn1 == "" || *dsn2 == "" {
		log.Fatal("dsn can not be empty")
	}

	e, err := executor.NewABTest(*dsn1, *dsn2, &executor.Option{
		Mute: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Check database", matchDB.FindString(*dsn1))

	tables, err := e.GetConn().FetchTables(matchDB.FindString(*dsn1))
	if err != nil {
		log.Fatal(err)
	}

	for _, table := range tables {
		log.Info("Check table", table)
		r1, err := e.GetConn1().Select(fmt.Sprintf(selectStmt, table))
		if err != nil {
			log.Fatal(err)
		}
		r2, err := e.GetConn2().Select(fmt.Sprintf(selectStmt, table))
		if err != nil {
			log.Fatal(err)
		}
		compare(r1, r2)
	}

	log.Info("Consistency check pass")
}

func compare(r1 [][]*connection.QueryItem, r2 [][]*connection.QueryItem) {
	if len(r1) != len(r2) {
		compareUUID(r1, r2)
	}
}

func compareUUID(r1 [][]*connection.QueryItem, r2 [][]*connection.QueryItem) {
	var (
		uuid1 = getUUIDColumn(r1)
		uuid2 = getUUIDColumn(r2)
	)

	if len(uuid1) < len(uuid2) {
		uuid1, uuid2 = uuid2, uuid1
	}

	for _, u1 := range uuid1 {
		found := false
		for _, u2 := range uuid2 {
			if u1 == u2 {
				found = true
			}
		}
		if !found {
			log.Infof("uuid %s not found", u1)
		}
	}
}

func getUUIDColumn(r [][]*connection.QueryItem) []string {
	uuid := []string{}
	for _, line := range r {
		for _, item := range line {
			if item.ValType.Name() == "uuid" {
				uuid = append(uuid, item.ValString)
			}
		}
	}
	return uuid
}
