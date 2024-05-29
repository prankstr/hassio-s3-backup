package backends

import (
	"bytes"
	"context"
	"fmt"
	"hassio-proton-drive-backup/models"
	"hassio-proton-drive-backup/pkg/services"
	"io"
	"log/slog"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	S3 = "S3"
)

type s3 struct {
	Client *minio.Client
	Bucket string
}

var _ models.StorageService = &s3{}

func NewS3Service(cs *services.ConfigService) (*s3, error) {
	s := s3{}
	s.Bucket = cs.GetS3BucketName()
	creds := credentials.NewStaticV4(cs.GetS3AccessKeyID(), cs.GetS3SecretAccessKey(), "")

	slog.Debug("Initializing S3 client", "endpoint", cs.GetS3Endpoint(), "bucket", s.Bucket)
	client, err := minio.New(cs.GetS3Endpoint(), &minio.Options{
		Creds:  creds,
		Secure: true,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create minio client: %v", err)
	}

	bucketExists, err := client.BucketExists(context.Background(), cs.GetS3BucketName())
	if err != nil {
		return nil, fmt.Errorf("could not check if bucket exists: %v", err)
	}

	if !bucketExists {
		err := client.MakeBucket(context.Background(), cs.GetS3BucketName(), minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("could not create bucket: %v", err)
		}
	}

	s.Client = client

	return &s, nil
}

func (s *s3) Login(creds *models.Credentials) error {
	return nil
}

func (s *s3) About() ([]byte, error) {
	return []byte("Storj"), nil
}

func (s *s3) UploadBackup(name string, path string) (string, error) {
	ctx := context.Background()
	contentType := "application/octet-stream"

	info, err := s.Client.FPutObject(ctx, s.Bucket, name, path, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("could not upload object: %v", err)
	}

	return info.Key, nil
}

func (s *s3) DownloadBackup(name string) (io.ReadCloser, error) {
	// Download the object.
	object, err := s.Client.GetObject(context.Background(), "mybucket", "myobject", minio.GetObjectOptions{})
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
	err := s.Client.RemoveObject(context.Background(), s.Bucket, name, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("could not delete object: %v", err)
	}

	return nil
}

func (s *s3) GetBackupAttributes(name string) (*models.FileAttributes, error) {
	// Open the object.
	object, err := s.Client.StatObject(context.Background(), s.Bucket, name, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not open object: %v", err)
	}

	return &models.FileAttributes{
		Size:     float64(object.Size) / (1024 * 1024), // convert bytes to MB
		Modified: object.LastModified,
	}, nil
}

func (s *s3) ListBackups() ([]*models.DirectoryItem, error) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	// List the objects in the bucket.
	var directoryData []*models.DirectoryItem

	objectCh := s.Client.ListObjects(ctx, s.Bucket, minio.ListObjectsOptions{})
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("could not list objects: %v", object.Err)
		}

		directoryData = append(directoryData, &models.DirectoryItem{
			Identifier: object.Key,
			Name:       strings.TrimSuffix(object.Key, ".tar"),
		})
	}

	return directoryData, nil
}

func (s *s3) FileExists(name string) bool {
	_, err := s.Client.StatObject(context.Background(), s.Bucket, name, minio.StatObjectOptions{})
	return err == nil
}
