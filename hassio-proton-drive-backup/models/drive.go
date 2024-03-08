package models

import (
	"io"
	"time"
)

// DirectoryData represents the details of a directory item
type DirectoryData struct {
	Identifier string `json:"Link"` // Proton Drive Link to file
	Name       string `json:"Name"` // Name of the item
}

// FileAttributes represents the attributes of a file
type FileAttributes struct {
	Size     float64   // Size of the backup in MB
	Modified time.Time // Time when the backup was last modifier(hopefully uploaded as well)
}

// Drive is an interface to represent a generic drive
type Drive interface {
	Login(username string, password string) error                   // Login to the drive
	About() ([]byte, error)                                         // Get information about the drive
	UploadFileByPath(name string, path string) (string, error)      // Upload a file to the drive
	DeleteFileByID(id string) error                                 // Delete a file from the drive
	DownloadFileByID(id string) (io.ReadCloser, error)              // Download a file from the drive
	GetBackupAttributesByID(linkID string) (*FileAttributes, error) // Get the attributes of a file
	ListDirectory(linkID string) ([]*DirectoryData, error)          // List the contents of a directory
	ListBackupDirectory() ([]*DirectoryData, error)                 // List the contents of the backup directory
	FileExists(linkID string) bool                                  // Check if a file exists
}
