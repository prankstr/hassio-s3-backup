package models

import (
	protonDriveAPI "github.com/henrybear327/Proton-API-Bridge"
)

type Drive interface {
	Login(username string, password string) error
	About() ([]byte, error)
	UploadFileByPath(name string, path string) (string, error)
	DeleteFileByID(id string) error
	ListDirectory(linkID string) ([]*protonDriveAPI.ProtonDirectoryData, error)
	ListBackupDirectory() ([]*protonDriveAPI.ProtonDirectoryData, error)
	FileExists(linkID string) bool
}
