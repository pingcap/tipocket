package loki

import (
	"fmt"
	"time"

	"github.com/grafana/loki/pkg/logcli/client"
	"github.com/grafana/loki/pkg/loghttp"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/pingcap/errors"
	"github.com/pingcap/log"
)

const (
	equalMatcher = "|="
	regexMatcher = "|~"
)

type LokiClient struct {
	cli       *client.Client
	startTime time.Time
}

// NewLokiClient creates a client to query loki.
func NewLokiClient(address, username, password string) *LokiClient {
	if address == "" {
		return nil
	}
	return &LokiClient{
		cli: &client.Client{
			Address:  address,
			Username: username,
			Password: password,
		},
		startTime: time.Now(),
	}
}

// FetchPodLogs gets pod logs from loki API. ns and podName are used
// to match the specific logs in loki. match is a string which you want to query
// from loki, you can set isRegex to true to make it to be a regex match. nonMatch
//  is a set of strings you don't want to match.
func (c *LokiClient) FetchPodLogs(ns, podName, match string, nonMatch []string, queryFrom, queryTo time.Time, isRegex bool) ([]string, error) {
	if ns == "" {
		return nil, errors.New("namespace must be set")
	}

	if podName == "" {
		return nil, errors.New("pod name must be set")
	}

	if match == "" {
		return nil, errors.New("match query must be set")
	}

	if !queryFrom.Before(queryTo) {
		return nil, errors.New("query to time must be after query from time")
	}

	if queryFrom.Before(c.startTime) {
		log.Info("query from time cannot be early than case start time. " +
			"set to case start time by default")
		queryFrom = c.startTime
	}

	var op string
	if isRegex {
		op = regexMatcher
	} else {
		op = equalMatcher
	}

	// Format the query to loki.
	query := fmt.Sprintf(`{instance="%s", namespace="%s"} %s"%s"`,
		podName, ns, op, match)

	var nonEqual string
	for _, v := range nonMatch {
		nonEqual += fmt.Sprintf(` !="%s"`, v)
	}
	query += nonEqual

	res, err := c.cli.QueryRange(query, 1000, queryFrom, queryTo, logproto.BACKWARD, 15*time.Second, true)
	if err != nil {
		return nil, err
	}

	var ret []string
	vals := res.Data.Result.(loghttp.Streams)
	for _, v := range vals {
		for _, entry := range v.Entries {
			ret = append(ret, entry.Line)
		}
	}

	return ret, nil
}
