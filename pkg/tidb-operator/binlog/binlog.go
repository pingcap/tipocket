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

package binlog

// NodeStatus represents the status saved in etcd.
type NodeStatus struct {
	NodeID string `json:"nodeId"`
	Host   string `json:"host"`
	State  string `json:"state"`

	// NB: Currently we save the whole `NodeStatus` in the status of the CR.
	// However, the following fields will be updated continuously.
	// To avoid CR being updated and re-synced continuously, we exclude these fields.
	// MaxCommitTS int64  `json:"maxCommitTS"`
	// UpdateTS    int64  `json:"updateTS"`
}
