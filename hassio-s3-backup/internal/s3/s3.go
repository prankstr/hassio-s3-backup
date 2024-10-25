package s3

import (
	"context"
	"fmt"
	"hassio-proton-drive-backup/internal/config"
	"log/slog"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Object struct {
	Modified time.Time `json:"modified"`
	Key      string    `json:"key"`
	Size     float64   `json:"size"`
}

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

// NewClient creates a new S3 client
func NewClient(cs *config.Service) (*minio.Client, error) {
	c := cs.Config
	// Get bucket and credentials from config
	bucket := c.S3.Bucket
	creds := credentials.NewStaticV4(c.S3.AccessKeyID, c.S3.SecretAccessKey, "")

	// Parse the S3 endpoint URL from the config
	url, err := url.Parse(c.S3.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("could not parse endpoint: %v", err)
	}

	// Determine if the connection should be secure based on the URL scheme
	isSecure := url.Scheme == "https"

	// Create minio options with the credentials and security settings
	opts := &minio.Options{
		Creds:  creds,
		Secure: isSecure,
	}

	// Log the initialization of the S3 client with debug level
	slog.Debug("initializing S3 client", "endpoint", url, "bucket", bucket)

	// Create a new minio client with the parsed URL host and options
	client, err := minio.New(url.Host, opts)
	if err != nil {
		return nil, fmt.Errorf("could not create S3 client: %v", err)
	}

	// Check if the specified bucket exists in S3
	bucketExists, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, fmt.Errorf("could not check if bucket exists: %v", err)
	}

	// If the bucket does not exist, create it
	if !bucketExists {
		err := client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("could not create bucket: %v", err)
		}
	}

	// Return the initialized S3 client
	return client, nil
}
