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

package dmutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pingcap/errors"

	httputil "github.com/pingcap/tipocket/pkg/util/http"
)

const (
	contentJSON = "application/json"
)

// Client is a HTTP Client for DM.
type Client struct {
	c         *httputil.Client
	urlPrefix string
}

// NewDMClient creates a HTTP Client for DM.
func NewDMClient(c *http.Client, masterAddr string) *Client {
	return &Client{
		c:         httputil.NewHTTPClient(c),
		urlPrefix: fmt.Sprintf("http://%s/apis/v1alpha1/", masterAddr),
	}
}

// CreateSource does `operate-source create` operation.
func (c *Client) CreateSource(conf string) error {
	input := map[string]interface{}{
		"op":     1, // "create"
		"config": conf,
	}
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	// sometimes may got 404 here, but I don't know why.
	res, err := c.c.Put(c.urlPrefix+"sources", contentJSON, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	// verify response content.
	ress := string(res)
	if strings.Count(ress, `"result":true`) != 2 {
		if !strings.Contains(ress, "already exists") {
			return errors.New(fmt.Sprintf("`operate-source create` failed, %s", ress))
		}
	}
	return nil
}

// StartTask does `start-task` operation.
func (c *Client) StartTask(conf string, sourceCount int) error {
	input := map[string]interface{}{
		"task": conf,
	}
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	res, err := c.c.Post(c.urlPrefix+"tasks", contentJSON, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	// verify response content.
	ress := string(res)
	if strings.Count(ress, `"result":true`) != sourceCount+1 {
		if !strings.Contains(ress, "already exist") {
			return errors.New(fmt.Sprintf("`start-task` failed, %s", ress))
		}
	}
	return nil
}

// CheckTaskStage checks task's current stage by `query-status`.
func (c *Client) CheckTaskStage(task, stage string, sourceCount int) error {
	res, err := c.c.Get(c.urlPrefix + "status/" + task)
	if err != nil {
		return err
	}

	// verify stage.
	ress := string(res)
	if strings.Count(ress, fmt.Sprintf(`"stage":"%s"`, stage)) != sourceCount {
		return errors.New(fmt.Sprintf("check task stage failed, %s", ress))
	}
	return nil
}
