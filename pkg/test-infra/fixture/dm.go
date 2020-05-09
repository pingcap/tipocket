// Copyright 2020 PingCAP, Inc.
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

package fixture

// DMConfig is the configuration for DM (Data Migration) component.
type DMConfig struct {
	MySQLConf     MySQLConfig // all MySQL instances use the same config now.
	DMVersion     string
	MasterReplica int // replicas of DM-master
	WorkerReplica int // replicas of DM-worker
	LogPath       string
}
