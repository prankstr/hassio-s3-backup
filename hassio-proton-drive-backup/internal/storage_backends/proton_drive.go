package storage_backends

import (
	"context"
	"encoding/json"
	"fmt"
	"hassio-proton-drive-backup/internal/config"
	"hassio-proton-drive-backup/internal/storage"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	protonDriveAPI "github.com/henrybear327/Proton-API-Bridge"

	"github.com/henrybear327/go-proton-api"
)

type protonDriveCredentials struct {
	UID           string
	AccessToken   string
	RefreshToken  string
	SaltedKeyPass string
}

type protonDriveService struct {
	drive       *protonDriveAPI.ProtonDrive
	backupLink  *proton.Link
	credentials *protonDriveCredentials
}

var _ storage.StorageService = &protonDriveService{}

// NewProtonDriveService initializes and returns a new HassioHandler
func NewProtonDriveService(cs *config.ConfigService) (*protonDriveService, error) {
	s := protonDriveService{}

	err := s.Login(cs.GetCredentials())
	if err != nil {
		return &s, err
	}

	config := cs.GetConfig()
	backupLink, err := s.drive.SearchByNameInActiveFolder(context.Background(), s.drive.RootLink, config.BackupDirectory, false, true, proton.LinkStateActive)
	if err != nil {
		return &protonDriveService{}, err
	}

	// Create backup dir if it doesn't exist
	if backupLink == nil {
		id, err := s.drive.CreateNewFolder(context.Background(), s.drive.RootLink, config.BackupDirectory)
		if err != nil {
			return &protonDriveService{}, err
		}

		backupLink, err = s.drive.GetLink(context.Background(), id)
		if err != nil {
			return &protonDriveService{}, err
		}
	}

	s.backupLink = backupLink

	return &s, nil
}

// NewProtonDrive initializes and returns a new protonDrive
func (s *protonDriveService) Login(creds *storage.Credentials) error {
	// Initialize ProtonDriveAPI configuration
	protonConf := protonDriveAPI.NewDefaultConfig()
	protonConf.ReplaceExistingDraft = true
	protonConf.AppVersion = "macos-drive@1.0.0-alpha.1"
	protonConf.FirstLoginCredential.Username = creds.Username
	protonConf.FirstLoginCredential.Password = creds.Password

	// Create a context for ProtonDriveAPI
	ctx := context.Background()

	// Check if credentials are set
	if s.credentials != nil {
		slog.Debug("Logging in with cached credentials")
		protonConf.UseReusableLogin = true

		protonConf.ReusableCredential.UID = s.credentials.UID
		protonConf.ReusableCredential.AccessToken = s.credentials.AccessToken
		protonConf.ReusableCredential.RefreshToken = s.credentials.RefreshToken
		protonConf.ReusableCredential.SaltedKeyPass = s.credentials.SaltedKeyPass

		protonDrive, _, err := protonDriveAPI.NewProtonDrive(ctx, protonConf, func(auth proton.Auth) {}, func() {})
		if err != nil {
			slog.Debug("Unable to login with cached credentials")

			// clear credentials on fail
			s.credentials = nil

			return err
		} else {
			slog.Debug("Used cached credentials to initialize the ProtonDrive")
			s.drive = protonDrive
			return nil
		}
	}

	// Initialize ProtonDrive
	slog.Debug("Logging in with username and password")
	protonDrive, pcreds, err := protonDriveAPI.NewProtonDrive(ctx, protonConf, func(auth proton.Auth) {}, func() {})
	if err != nil {
		return err
	}

	s.credentials = &protonDriveCredentials{}
	s.credentials.UID = pcreds.UID
	s.credentials.AccessToken = pcreds.AccessToken
	s.credentials.RefreshToken = pcreds.RefreshToken
	s.credentials.SaltedKeyPass = pcreds.SaltedKeyPass

	s.drive = protonDrive
	return nil
}

// FileExists returns true if a file exists
func (s *protonDriveService) FileExists(linkID string) bool {
	ctx := context.Background()

	link, err := s.drive.GetLink(ctx, linkID)
	if err != nil {
		fmt.Println("Couldn't get link from linkID", err)
		return false
	}

	_, err = s.drive.GetRevisions(ctx, link, 0)
	if err != nil {
		if strings.Contains(err.Error(), "File or folder was not found") && strings.Contains(err.Error(), "Code=2501") {
			return true
		}

		fmt.Println("Couldn't get file revision: ", err)
		return false
	}

	return true
}

// About returns information about the drive
func (s *protonDriveService) About() ([]byte, error) {
	// Access h.ProtonDriveService for /ProtonDrive logic
	res, _ := s.drive.About(context.Background())

	// Convert response to JSON
	data, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// UploadFileByPath uploads a file to the drive
func (s *protonDriveService) UploadBackup(name string, path string) (string, error) {
	// Create a new file
	linkID, _, err := s.drive.UploadFileByPath(context.Background(), s.backupLink, name, path, 0)
	if err != nil {
		return "", err
	}

	return linkID, nil
}

// DeleteFileByID deletes a file from the drive
func (s *protonDriveService) DeleteBackup(id string) error {
	// Delete the backup
	err := s.drive.MoveFileToTrashByID(context.Background(), id)
	if err != nil {
		return err
	}

	return nil
}

// DownloadFileByID downloads a file from the drive
func (s *protonDriveService) DownloadBackup(id string) (io.ReadCloser, error) {
	reader, _, _, err := s.drive.DownloadFileByID(context.Background(), id, 0)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

// GetBackupAttributesByID returns the attributes of a file
func (s *protonDriveService) GetBackupAttributes(id string) (*storage.FileAttributes, error) {
	protonAttributes, err := s.drive.GetActiveRevisionAttrsByID(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return &storage.FileAttributes{
		Size:     float64(protonAttributes.Size) / (1024 * 1024), // convert bytes to MB
		Modified: protonAttributes.ModificationTime,
	}, nil
}

// ListBackupDirectory returns a list items in the backup directory
func (s *protonDriveService) ListBackups() ([]*storage.DirectoryItem, error) {
	items, err := s.drive.ListDirectory(context.Background(), s.backupLink.LinkID)
	if err != nil {
		return nil, err
	}

	var protonBackups []*storage.DirectoryItem
	for _, item := range items {
		if item.IsFolder {
			continue
		}

		protonBackups = append(protonBackups, &storage.DirectoryItem{
			Identifier: item.Link.LinkID,
			Name:       strings.TrimSuffix(item.Name, filepath.Ext(item.Name)),
		})
	}

	return protonBackups, nil
}
