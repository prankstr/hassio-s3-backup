package services

import (
	"fmt"
	"hassio-proton-drive-backup/models"
)

type SyncService struct {
	drive models.Drive
}

func NewSyncService(drive models.Drive) *SyncService {
	return &SyncService{
		drive: drive,
	}
}

func (s SyncService) Sync(backup *models.Backup) (string, error) {
	path := fmt.Sprintf("%s/%s.%s", "/backup", backup.HA.Slug, "tar")

	fmt.Println("Syncing backup to ProtonDrive: ", backup.Name, " from path: ", path)
	backup.Status = models.StatusSyncing
	id, err := s.drive.UploadFileByPath(fmt.Sprintf("%s.%s", backup.Name, ".tar"), path)
	if err != nil {
		backup.Status = models.StatusFailed
		fmt.Println(err)
		return "", err
	}

	return id, nil
}
