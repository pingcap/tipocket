package loki

import (
	"fmt"
	"time"

	"github.com/grafana/loki/pkg/logcli/client"
	"github.com/grafana/loki/pkg/loghttp"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/pingcap/errors"
)

const (
	equalMatcher    = "|="
	regexMatcher    = "|~"
)

type lokiClient struct {
	cli       *client.Client
	startTime time.Time
}

// NewLokiClient creates a client to query loki.
func NewLokiClient(address, username, password string) *lokiClient {
	return &lokiClient{
		cli: &client.Client{
			Address:  address,
			Username: username,
			Password: password,
		},
		startTime: time.Now(),
	}
}

// FetchContainerLogs gets logs from loki API. ns and containerName are used
// to match the specific logs in loki. match is a string which you want to query
// from loki, you can set isRegex to true to make it to be a regex match. nonMatch
//  is a set of strings you don't want to match. lookbackTime is the query duration.
func (c *lokiClient) FetchContainerLogs(ns, containerName, match string, nonMatch []string, lookbackTime time.Duration, isRegex bool) ([]string, int, error) {
	if ns == "" {
		return nil, 0, errors.New("namespace must be set")
	}

	if containerName == "" {
		return nil, 0, errors.New("container name must be set")
	}

	if match == "" {
		return nil, 0, errors.New("match query must be set")
	}

	var op string
	if isRegex {
		op = regexMatcher
	} else {
		op = equalMatcher
	}

	// Format the logql query for loki.
	query := fmt.Sprintf(`{container_name="%s", namespace="%s"} %s"%s"`,
		containerName, ns, op, match)

	var nonEqual string
	for _, v := range nonMatch {
		nonEqual += fmt.Sprintf(` !="%s"`, v)
	}
	query += nonEqual

	res, err := c.cli.QueryRange(query, 1000, time.Now().Truncate(lookbackTime), time.Now(), logproto.BACKWARD, 15*time.Second, true)
	if err != nil {
		return nil, 0, err
	}

	var ret []string
	vals := res.Data.Result.(loghttp.Streams)
	for _, v := range vals {
		for _, entry := range v.Entries {
			ret = append(ret, entry.Line)
		}
	}

	return ret, len(ret), nil
}
