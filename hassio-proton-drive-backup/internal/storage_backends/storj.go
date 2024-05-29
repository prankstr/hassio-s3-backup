package storage_backends

import (
	"bytes"
	"context"
	"fmt"
	"hassio-proton-drive-backup/internal/config"
	"hassio-proton-drive-backup/internal/storage"
	"io"
	"os"
	"strings"

	"storj.io/uplink"
)

type storjService struct {
	project *uplink.Project
	bucket  *uplink.Bucket
	access  *uplink.Access
}

var _ storage.StorageService = &storjService{}

func NewStorjService(cs *config.ConfigService) (*storjService, error) {
	s := storjService{}
	ctx := context.Background()

	// Parse access grant, which contains necessary credentials and permissions.
	err := s.Login(cs.GetCredentials())
	if err != nil {
		return &storjService{}, err
	}

	// Open up the Project we will be working with.
	project, err := uplink.OpenProject(ctx, s.access)
	if err != nil {
		return &storjService{}, fmt.Errorf("could not open project: %v", err)
	}
	defer project.Close()

	// Ensure the desired Bucket within the Project is created.
	bucket, err := project.EnsureBucket(ctx, cs.GetStorjBucketName())
	if err != nil {
		return &storjService{}, fmt.Errorf("could not ensure bucket: %v", err)
	}

	s.project = project
	s.bucket = bucket

	return &s, nil
}

func (s *storjService) Login(creds *storage.Credentials) error {
	// Parse access grant, which contains necessary credentials and permissions.
	access, err := uplink.ParseAccess(creds.AccessGrant)
	if err != nil {
		return fmt.Errorf("could not get access grant: %v", err)
	}

	s.access = access

	return nil
}

func (s *storjService) About() ([]byte, error) {
	return []byte("Storj"), nil
}

func (s *storjService) UploadBackup(name string, path string) (string, error) {
	// Upload the file.
	upload, err := s.project.UploadObject(context.Background(), s.bucket.Name, name, &uplink.UploadOptions{})
	if err != nil {
		return "", fmt.Errorf("could not upload object: %v", err)
	}

	dataToUpload, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("could not read file: %v", err)
	}

	// Copy the data to the upload.
	buf := bytes.NewBuffer(dataToUpload)
	_, err = io.Copy(upload, buf)
	if err != nil {
		_ = upload.Abort()
		return "", fmt.Errorf("could not upload data: %v", err)
	}

	// Commit the uploaded object.
	err = upload.Commit()
	if err != nil {
		return "", fmt.Errorf("could not commit uploaded object: %v", err)
	}

	return name, nil
}

func (s *storjService) DownloadBackup(name string) (io.ReadCloser, error) {
	// Open the object.
	download, err := s.project.DownloadObject(context.Background(), s.bucket.Name, name, &uplink.DownloadOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not download object: %v", err)
	}
	defer download.Close()

	// Read everything from the download stream
	receivedContents, err := io.ReadAll(download)
	if err != nil {
		return nil, fmt.Errorf("could not read data: %v", err)
	}

	readCloser := io.NopCloser(bytes.NewReader(receivedContents))

	return readCloser, nil
}

func (s *storjService) DeleteBackup(name string) error {
	// Delete the object.
	_, err := s.project.DeleteObject(context.Background(), s.bucket.Name, name)
	if err != nil {
		return fmt.Errorf("could not delete object: %v", err)
	}
	return nil
}

func (s *storjService) GetBackupAttributes(name string) (*storage.FileAttributes, error) {
	// Open the object.
	object, err := s.project.StatObject(context.Background(), s.bucket.Name, name)
	if err != nil {
		return nil, fmt.Errorf("could not open object: %v", err)
	}

	return &storage.FileAttributes{
		Size:     float64(object.System.ContentLength) / (1024 * 1024), // convert bytes to MB
		Modified: object.System.Created,
	}, nil
}

func (s *storjService) ListBackups() ([]*storage.DirectoryItem, error) {
	// List the objects in the bucket.
	objects := s.project.ListObjects(context.Background(), s.bucket.Name, &uplink.ListObjectsOptions{})
	var directoryData []*storage.DirectoryItem
	for objects.Next() {
		object := objects.Item()
		directoryData = append(directoryData, &storage.DirectoryItem{
			Identifier: object.Key,
			Name:       strings.TrimSuffix(object.Key, ".tar"),
		})
	}
	return directoryData, nil
}

func (s *storjService) FileExists(name string) bool {
	_, err := s.project.StatObject(context.Background(), s.bucket.Name, name)
	return err == nil
}
