package models

import "time"

type ProtonDirectoryData struct {
	Link string `json:"Link"` // Proton Drive Link to file
	Name string `json:"Name"` // Name of the backup
}

type FileAttributes struct {
	Size     float64   // Size of the backup in MB
	Modified time.Time // Time when the backup was last modifier(hopefully uploaded as well)
}

type Drive interface {
	Login(username string, password string) error
	About() ([]byte, error)
	UploadFileByPath(name string, path string) (string, error)
	DeleteFileByID(id string) error
	GetBackupAttributesByID(linkID string) (*FileAttributes, error)
	ListDirectory(linkID string) ([]*ProtonDirectoryData, error)
	ListBackupDirectory() ([]*ProtonDirectoryData, error)
	FileExists(linkID string) bool
}
