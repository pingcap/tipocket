package artifacts

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/pingcap/tipocket/pkg/cluster/manager/util"
)

// S3Client encapsulate minio-go sdk
type S3Client struct {
	*minio.Client
}

// NewS3Client creates an S3client instance
func NewS3Client() (*S3Client, error) {
	minioClient, err := minio.New(util.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(util.AwsAccessKeyID, util.AwsSecretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}
	return &S3Client{minioClient}, nil
}
