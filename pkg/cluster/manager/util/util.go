package util

import (
	"fmt"
	"math/rand"

	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

// Addr specifies api_server listening address
var Addr string

// S3Endpoint specifies s3 object storage address
var S3Endpoint string

// AwsAccessKeyID specifies S3 access key id
var AwsAccessKeyID string

// AwsSecretAccessKey specifies S3 secret access key
var AwsSecretAccessKey string

// RandomResource ...
func RandomResource(rs []*types.Resource) (*types.Resource, error) {
	if len(rs) == 0 {
		return nil, fmt.Errorf("expect non-empty resources")
	}
	return rs[rand.Intn(len(rs))], nil
}
