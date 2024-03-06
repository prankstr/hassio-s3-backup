package models

type ProtonDirectoryData struct {
	Link string
	Name string
}

type Drive interface {
	Login(username string, password string) error
	About() ([]byte, error)
	UploadFileByPath(name string, path string) (string, error)
	DeleteFileByID(id string) error
	ListDirectory(linkID string) ([]*ProtonDirectoryData, error)
	ListBackupDirectory() ([]*ProtonDirectoryData, error)
	FileExists(linkID string) bool
}
