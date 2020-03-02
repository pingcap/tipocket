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

package br

import "sigs.k8s.io/controller-runtime/pkg/client"

// TODO: impl
// BrOps knows how to operate BR on k8s
type BrOps struct {
	cli client.Client
}

func New(cli client.Client) *BrOps {
	return &BrOps{cli}
}
