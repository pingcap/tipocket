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

package executor

// Option struct
type Option struct {
	ID         int
	Clear      bool
	Log        string
	LogSuffix  string
	Reproduce  string
	Stable     bool
	Mute       bool
	OnlineDDL  bool
	GeneralLog bool
	Hint       bool
	TiFlash    bool
}

// Clone option
func (o *Option) Clone() *Option {
	o1 := *o
	return &o1
}
