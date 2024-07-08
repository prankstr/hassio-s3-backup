package s3

import (
	"context"
	"fmt"
	"hassio-proton-drive-backup/internal/config"
	"log/slog"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Object struct {
	Modified time.Time
	Key      string `json:"key"`
	Size     float64
}

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

func NewClient(cs *config.Service) (*minio.Client, error) {
	bucket := cs.GetS3Bucket()
	creds := credentials.NewStaticV4(cs.GetS3AccessKeyID(), cs.GetS3SecretAccessKey(), "")

	slog.Debug("Initializing S3 client", "endpoint", cs.GetS3Endpoint(), "bucket", bucket)
	client, err := minio.New(cs.GetS3Endpoint(), &minio.Options{
		Creds:  creds,
		Secure: true,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create minio client: %v", err)
	}

	bucketExists, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, fmt.Errorf("could not check if bucket exists: %v", err)
	}

	if !bucketExists {
		err := client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("could not create bucket: %v", err)
		}
	}

	return client, nil
}
