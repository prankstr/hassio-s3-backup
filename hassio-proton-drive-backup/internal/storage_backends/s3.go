package storage_backends

import (
	"bytes"
	"context"
	"fmt"
	"hassio-proton-drive-backup/internal/config"
	"hassio-proton-drive-backup/internal/storage"
	"io"
	"log/slog"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type s3 struct {
	client *minio.Client
	creds  *storage.Credentials
	bucket string
}

var _ storage.Service = &s3{}

func NewS3Service(cs *config.Service) (*s3, error) {
	s := s3{}
	s.bucket = cs.GetS3Bucket()
	creds := credentials.NewStaticV4(cs.GetS3AccessKeyID(), cs.GetS3SecretAccessKey(), "")

	slog.Debug("Initializing S3 client", "endpoint", cs.GetS3Endpoint(), "bucket", s.bucket)
	client, err := minio.New(cs.GetS3Endpoint(), &minio.Options{
		Creds:  creds,
		Secure: true,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create minio client: %v", err)
	}

	bucketExists, err := client.BucketExists(context.Background(), s.bucket)
	if err != nil {
		return nil, fmt.Errorf("could not check if bucket exists: %v", err)
	}

	if !bucketExists {
		err := client.MakeBucket(context.Background(), s.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("could not create bucket: %v", err)
		}
	}

	s.client = client

	return &s, nil
}

func (s *s3) Login() error {
	return nil
}

func (s *s3) About() ([]byte, error) {
	return []byte("Storj"), nil
}

func (s *s3) UploadBackup(name string, path string) (string, error) {
	ctx := context.Background()
	contentType := "application/octet-stream"

	info, err := s.client.FPutObject(ctx, s.bucket, name, path, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("could not upload object: %v", err)
	}

	return info.Key, nil
}

func (s *s3) DownloadBackup(name string) (io.ReadCloser, error) {
	// Download the object.
	object, err := s.client.GetObject(context.Background(), "mybucket", "myobject", minio.GetObjectOptions{})
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("could not get object: %v", err)
	}
	defer object.Close()

	// Read everything from the download stream
	receivedContents, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("could not read data: %v", err)
	}

	readCloser := io.NopCloser(bytes.NewReader(receivedContents))

	return readCloser, nil
}

func (s *s3) DeleteBackup(name string) error {
	// Delete the object.
	err := s.client.RemoveObject(context.Background(), s.bucket, name, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("could not delete object: %v", err)
	}

	return nil
}

func (s *s3) GetBackupAttributes(name string) (*storage.FileAttributes, error) {
	// Open the object.
	object, err := s.client.StatObject(context.Background(), s.bucket, name, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not open object: %v", err)
	}

	return &storage.FileAttributes{
		Size:     float64(object.Size) / (1024 * 1024), // convert bytes to MB
		Modified: object.LastModified,
	}, nil
}

func (s *s3) ListBackups() ([]*storage.DirectoryItem, error) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	// List the objects in the bucket.
	var directoryData []*storage.DirectoryItem

	objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{})
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("could not list objects: %v", object.Err)
		}

		directoryData = append(directoryData, &storage.DirectoryItem{
			Identifier: object.Key,
			Name:       strings.TrimSuffix(object.Key, ".tar"),
		})
	}

	return directoryData, nil
}

func (s *s3) FileExists(name string) bool {
	_, err := s.client.StatObject(context.Background(), s.bucket, name, minio.StatObjectOptions{})
	return err == nil
}
