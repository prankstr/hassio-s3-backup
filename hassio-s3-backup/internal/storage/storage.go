package storage

import (
	"io"
	"time"
)

const (
	S3     = "S3"
	STORJ  = "Storj"
	PROTON = "Proton Drive"
)

// DirectoryItem represents the details of a directory item
type DirectoryItem struct {
	Identifier string `json:"identifier"` // unique identifier of the item on the storage backend
	Name       string `json:"name"`       // Name of the item
}

// FileAttributes represents the attributes of a file
type FileAttributes struct {
	Modified time.Time // Last modified time
	Size     float64   // Size in MB
}

type Credentials struct {
	Username        string
	Password        string
	AccessGrant     string
	AccessKeyID     string
	SecretAccessKey string
}

// Drive is an interface to represent a generic drive
type Service interface {
	Login() error                                          // Login to the drive
	About() ([]byte, error)                                // Get information about the drive
	UploadBackup(name string, path string) (string, error) // Upload a file to the drive
	DeleteBackup(id string) error                          // Delete a file from the drive
	DownloadBackup(id string) (io.ReadCloser, error)       // Download a file from the drive
	// DownloadBackup(id string) error                             // Download a file from the drive
	GetBackupAttributes(linkID string) (*FileAttributes, error) // Get the attributes of a file
	ListBackups() ([]*DirectoryItem, error)                     // List the contents of the backup directory
	FileExists(linkID string) bool                              // Check if a file exists
}
