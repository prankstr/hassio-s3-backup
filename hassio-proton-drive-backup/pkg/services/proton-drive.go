package services

import (
	"context"
	"encoding/json"
	"fmt"
	"hassio-proton-drive-backup/models"
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

var _ models.Drive = &protonDriveService{}

// NewProtonDriveService initializes and returns a new HassioHandler
func NewProtonDriveService(cs *ConfigService) (protonDriveService, error) {
	s := protonDriveService{}

	err := s.Login(cs.GetProtonDriveUser(), cs.GetProtonDrivePassword())
	if err != nil {
		return protonDriveService{}, err
	}

	config := cs.GetConfig()
	backupLink, err := s.drive.SearchByNameInActiveFolder(context.Background(), s.drive.RootLink, config.BackupDirectory, false, true, proton.LinkStateActive)
	if err != nil {
		return protonDriveService{}, err
	}

	// Create backup dir if it doesn't exist
	if backupLink == nil {
		id, err := s.drive.CreateNewFolder(context.Background(), s.drive.RootLink, config.BackupDirectory)
		if err != nil {
			return protonDriveService{}, err
		}

		backupLink, err = s.drive.GetLink(context.Background(), id)
		if err != nil {
			return protonDriveService{}, err
		}
	}

	s.backupLink = backupLink

	return s, nil
}

// NewProtonDrive initializes and returns a new protonDrive
func (s *protonDriveService) Login(username string, password string) error {
	// Initialize ProtonDriveAPI configuration
	protonConf := protonDriveAPI.NewDefaultConfig()
	protonConf.ReplaceExistingDraft = true
	protonConf.AppVersion = "macos-drive@1.0.0-alpha.1+rclone"
	protonConf.FirstLoginCredential.Username = username
	protonConf.FirstLoginCredential.Password = password

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
	slog.Debug("Logging in with username and passwordß")
	protonDrive, creds, err := protonDriveAPI.NewProtonDrive(ctx, protonConf, func(auth proton.Auth) {}, func() {})
	if err != nil {
		return err
	}

	s.credentials = &protonDriveCredentials{}
	s.credentials.UID = creds.UID
	s.credentials.AccessToken = creds.AccessToken
	s.credentials.RefreshToken = creds.RefreshToken
	s.credentials.SaltedKeyPass = creds.SaltedKeyPass

	s.drive = protonDrive
	return nil
}

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

func (s *protonDriveService) UploadFileByPath(name string, path string) (string, error) {
	// Create a new file
	linkID, _, err := s.drive.UploadFileByPath(context.Background(), s.backupLink, name, path, 0)
	if err != nil {
		return "", err
	}

	return linkID, nil
}

func (s *protonDriveService) DeleteFileByID(id string) error {
	// Delete the backup
	err := s.drive.MoveFileToTrashByID(context.Background(), id)
	if err != nil {
		return err
	}

	return nil
}

func (s *protonDriveService) GetBackupAttributesByID(id string) (*models.FileAttributes, error) {
	protonAttributes, err := s.drive.GetActiveRevisionAttrsByID(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return &models.FileAttributes{
		Size:     float64(protonAttributes.Size) / (1024 * 1024), //convert bytes to MB
		Modified: protonAttributes.ModificationTime,
	}, nil
}

func (s *protonDriveService) ListBackupDirectory() ([]*models.DirectoryData, error) {
	items, err := s.drive.ListDirectory(context.Background(), s.backupLink.LinkID)
	if err != nil {
		return nil, err
	}

	var protonBackups []*models.DirectoryData
	for _, item := range items {
		if item.IsFolder {
			continue
		}

		protonBackups = append(protonBackups, &models.DirectoryData{
			Identifier: item.Link.LinkID,
			Name:       strings.TrimSuffix(item.Name, filepath.Ext(item.Name)),
		})
	}

	return protonBackups, nil
}

func (s *protonDriveService) ListDirectory(linkID string) ([]*models.DirectoryData, error) {
	items, err := s.drive.ListDirectory(context.Background(), linkID)
	if err != nil {
		return nil, err
	}

	var protonBackups []*models.DirectoryData
	for _, item := range items {
		if item.IsFolder {
			continue
		}

		protonBackups = append(protonBackups, &models.DirectoryData{
			Identifier: item.Link.LinkID,
			Name:       strings.TrimSuffix(item.Name, filepath.Ext(item.Name)),
		})
	}

	return protonBackups, nil
}
