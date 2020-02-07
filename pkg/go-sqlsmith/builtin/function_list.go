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

package builtin

import "log"

var funcLists = [][]*functionClass{
	commonFunctions,
	mathFunctions,
	datetimeFunctions,
	stringFunctions,
	informationFunctions,
	controlFunctions,
	miscellaneousFunctions,
	encryptionFunctions,
	jsonFunctions,
	tidbFunctions,
}

func getFuncMap() map[string]*functionClass {
	funcs := make(map[string]*functionClass)
	for _, funcSet := range funcLists {
		for _, fn := range funcSet {
			if _, ok := funcs[fn.name]; ok {
				log.Fatalf("duplicated func name %s", fn.name)
			} else {
				funcs[fn.name] = fn
			}
		}
	}
	return funcs
}
