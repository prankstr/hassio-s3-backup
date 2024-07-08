package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hassio-proton-drive-backup/internal/config"
	"hassio-proton-drive-backup/internal/hassio"
	"hassio-proton-drive-backup/internal/s3"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
)

// BackupStatus is a custom type to represent the status of the backup
type Status string

const (
	StatusDeleting Status = "DELETING" // Backup in being deleted
	StatusPending  Status = "PENDING"  // Backup is initalized but no action taken
	StatusRunning  Status = "RUNNING"  // Backup is being created in Home Assistant
	StatusSynced   Status = "SYNCED"   // Backup is present in both Home Assistant and drive
	StatusHAOnly   Status = "HAONLY"   // Backup is only present in  Home Assistant
	StatusS3Only   Status = "S3ONLY"   // Backup is only present in S3
	StatusSyncing  Status = "SYNCING"  // Backups is being uploaded to drive
	StatusFailed   Status = "FAILED"   // Backup process failed somewhere
)

// Backup represents the details and status of a backup process
type Backup struct {
	Date         time.Time      `json:"date"`
	S3           *s3.Object     `json:"s3"`
	HA           *hassio.Backup `json:"ha"`
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Status       Status         `json:"status"`
	Slug         string         `json:"slug"`
	ErrorMessage string         `json:"errorMessage"`
	Size         float64        `json:"size"`
	KeepInHA     bool           `json:"keepInHA"`
	KeepInS3     bool           `json:"keepInS3"`
	Pinned       bool           `json:"pinned"`
}

// HAData is a selection of metadata from the HA backups
type HAData struct {
	Date time.Time `json:"date"`
	Slug string    `json:"slug"`
	Size float64   `json:"size"`
}

type Request struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProtonLink string `json:"protonLink"` // Proton Drive Link to file
	Slug       string `json:"slug"`       // Backup slug from Home assistant
}

// UpdateStatus updates the status of the backup
func (b *Backup) UpdateStatus(status Status) {
	b.Status = status
}

type Service struct {
	nextBackupCalculatedAt time.Time
	s3                     *minio.Client
	stopBackupChan         chan struct{}
	configService          *config.Service
	config                 *config.Options
	syncTicker             *time.Ticker
	ongoingBackups         map[string]struct{}
	stopSyncChan           chan struct{}
	backupTimer            *time.Timer
	hassio                 *hassio.Client
	backups                []*Backup
	backupsInS3            int
	backupsInHA            int
	nextBackupIn           time.Duration
	backupInterval         time.Duration
	syncInterval           time.Duration
	mutex                  sync.Mutex
}

func NewService(s3 *minio.Client, configService *config.Service) *Service {
	config := configService.Config

	hassioService := hassio.NewService(config.SupervisorToken)

	service := &Service{
		hassio:        hassioService,
		s3:            s3,
		configService: configService,
		config:        config,

		stopBackupChan: make(chan struct{}),
		stopSyncChan:   make(chan struct{}),
		backupInterval: configService.GetBackupInterval(),
		backupsInHA:    configService.GetBackupsInHA(),
		backupsInS3:    configService.GetBackupsInS3(),
		syncInterval:   1 * time.Minute, // set the interval for sync

		ongoingBackups: make(map[string]struct{}),
	}

	// Initial load and sync of backups
	service.loadBackupsFromFile()
	service.syncBackups()

	// Initialize backupTimer with a dummy duration
	service.backupTimer = time.NewTimer(time.Hour)
	service.backupTimer.Stop()

	// Start scheduled backups and syncs
	go service.startBackupScheduler()
	go service.startBackupSyncScheduler()
	go service.listenForConfigChanges(configService.ConfigChangeChan)

	return service
}

// PerformBackup creates a new backup and uploads it to the remote drive
func (s *Service) PerformBackup(name string) error {
	backup := s.initializeBackup(name)

	// Track ongoing backups to avoid syncing or any other manipulation in the meantime
	s.ongoingBackups[backup.ID] = struct{}{}

	slog.Info("Backup started", "name", backup.Name)
	slog.Info("Requesting backup from Home Assistant", "name", backup.Name)

	// Create backup in Home Assistant
	backup.UpdateStatus(StatusRunning)
	slug, err := s.requestHomeAssistantBackup(backup.Name)
	if err != nil {
		slog.Error("Failed to request backup from Home Assistant", "BackupName", backup.Name, "Error", err)
		backup.UpdateStatus(StatusFailed)
		s.removeOngoingBackup(backup.ID)
		return err
	}
	backup.Slug = slug

	if err := s.saveBackupsToFile(); err != nil {
		slog.Error("Error saving backup state after Home Assistant request", "name", backup.Name, "error", err)
		return err
	}

	// Update backup with HA Data and upload to Drive
	if err := s.processAndUploadBackup(backup); err != nil {
		slog.Error("Error syncing backup to S3", "name", backup.Name, "error", err)
		backup.UpdateStatus(StatusFailed)
		s.removeOngoingBackup(backup.ID)
		return err
	}

	// Remove ongoing backup and save state
	backup.UpdateStatus(StatusSynced)
	s.removeOngoingBackup(backup.ID)
	slog.Info("Backup process completed successfully", "BackupName", backup.Name, "Status", backup.Status)

	if err := s.saveBackupsToFile(); err != nil {
		slog.Error("Error saving backup state after syncing", "name", backup.Name, "error", err)
		return err
	}

	// Reset timer
	s.resetTimerForNextBackup()

	// Perform a sync after the backup to ensure state is up to date
	if err := s.syncBackups(); err != nil {
		slog.Error("Error syncing backups", "error", err)
	}

	return nil
}

// DeleteBackup deletes a backup from all sources
func (s *Service) DeleteBackup(id string) error {
	backupToDelete, deleteIndex := s.findBackupToDelete(id)
	if backupToDelete == nil {
		slog.Error("Backup not found for deletion", "id", id)
		return nil // or return an error indicating that the backup was not found
	}

	slog.Info("Initiating backup deletion", "Backup", backupToDelete.Name, "BackupID", backupToDelete.ID)
	backupToDelete.UpdateStatus(StatusDeleting)

	// Delete backup from Home Assistant
	slog.Debug("Deleting backup from Home Assistant", "ID", backupToDelete.ID)
	if err := s.deleteBackupInHomeAssistant(backupToDelete); err != nil {
		slog.Error("Failed to delete backup in Home Assistant", "Backup", backupToDelete.Name, "error", err)
	} else {
		slog.Debug("Backup deleted from Home Assistant", "name", backupToDelete.Name)
	}

	// Delete backup from the S3
	if backupToDelete.S3 != nil {
		slog.Debug("Deleting backup from S3", "ID", backupToDelete.ID)
		if err := s.deleteBackupFromS3(backupToDelete); err != nil {
			slog.Error("Failed to delete backup from S3 backend", "Backup", backupToDelete.Name, "Error", err)
		} else {
			slog.Debug("Backup deleted from S3", "name", backupToDelete.Name)
		}
	}

	// Delete backup from local "DB"
	if deleteIndex != -1 {
		slog.Debug("Removing backup from slice", "Index", deleteIndex, "ID", backupToDelete.ID)
		s.backups = append(s.backups[:deleteIndex], s.backups[deleteIndex+1:]...)
		slog.Debug("Backup removed from slice", "ID", backupToDelete.ID)
	}

	slog.Debug("Saving backup state after deletion", "ID", backupToDelete.ID)
	if err := s.saveBackupsToFile(); err != nil {
		slog.Error("Error saving backup state after deletion", "error", err)
		return err
	}

	s.resetTimerForNextBackup()

	slog.Debug("Backup deletion process completed", "name", backupToDelete.Name)
	return nil
}

// RestoreBackup calls home assistant to restore a backup
// Note: might not be needed, as the restore can be done from the Home Assistant UI
func (s *Service) RestoreBackup(id string) error {
	var backupToRestore *Backup

	for _, backup := range s.backups {
		if backup.ID == id {
			backupToRestore = backup
			break
		}
	}

	slog.Info("Attempting to restore to backup", "name", backupToRestore.Name)
	err := s.hassio.RestoreBackup(backupToRestore.HA.Slug)
	if err != nil {
		return fmt.Errorf("failed to restore backup in Home Assistant: %v", err)
	}

	slog.Info("Restored to backup", "name", backupToRestore.Name)
	return nil
}

// DownloadBackup downloads a backup to the specified directory
// Note: might not be needed, download can be done manually from drive and then uploaded to Home Assistant
func (s *Service) DownloadBackup(id string) error {
	var backup *Backup

	for _, b := range s.backups {
		if b.ID == id {
			backup = b
			break
		}
	}

	if backup == nil || backup.Slug == "" {
		slog.Error("The addon doesn't have the necessary information about the backup, please upload it manually to Home Assistant", "backup", backup.Name)
		return errors.New("the addon doesn't have the necessary information about the backup, please upload it manually to home assistant")
	}

	slog.Debug("Downloading backup to Home Assistant", "backup", backup.Name)
	backup.UpdateStatus(StatusSyncing)

	filePath := filepath.Join("/backup", backup.Name+".tar")
	err := s.s3.FGetObject(context.Background(), s.config.S3.Bucket, backup.Name, filePath, minio.GetObjectOptions{})
	if err != nil {
		slog.Error("Failed to write backup to disk", "backup", backup.Name, "filePath", filePath, "error", err)
		backup.UpdateStatus(StatusS3Only)
		return err
	}

	slog.Info("Backup downloaded successfully", "backup", backup.Name, "filePath", filePath)
	backup.KeepInHA = true
	backup.UpdateStatus(StatusSynced)

	return nil
}

// PinBackup pins a backup to prevent it from being deleted
func (s *Service) PinBackup(id string) error {
	for _, backup := range s.backups {
		if backup.ID == id {
			backup.KeepInHA = true
			backup.KeepInS3 = true
			backup.Pinned = true
			slog.Info("Backup pinned", "name", backup.Name)
			return s.saveBackupsToFile()
		}
	}

	return errors.New("backup not found")
}

// UninBackup unpins a backup to allow it to be deleted
func (s *Service) UnpinBackup(id string) error {
	for _, backup := range s.backups {
		if backup.ID == id {
			backup.Pinned = false
			slog.Info("Backup unpinned", "name", backup.Name)
			return s.saveBackupsToFile()
		}
	}

	return errors.New("backup not found")
}

// List Backups returns the addons list of backups in memory
func (s *Service) ListBackups() []*Backup {
	return s.backups
}

// TimeUntilNextBackup returns the time until next backup in milliseconds
func (s *Service) TimeUntilNextBackup() int64 {
	return time.Until(s.nextBackupCalculatedAt.Add(s.nextBackupIn)).Milliseconds()
}

// NameExists checks if a backup with the given name exists
func (s *Service) NameExists(name string) bool {
	generatedName := s.generateBackupName(name)

	for _, backup := range s.backups {
		if backup.Name == generatedName {
			return true
		}
	}

	return false
}

// initializeBackup return a new internal backup object
func (s *Service) initializeBackup(name string) *Backup {
	backup := &Backup{
		ID:       s.generateBackupID(),
		Name:     s.generateBackupName(name),
		Date:     time.Now().In(s.config.Timezone),
		Status:   StatusPending,
		KeepInHA: true,
		KeepInS3: true,
		S3:       nil,
		HA:       nil,
	}

	s.backups = append([]*Backup{backup}, s.backups...)

	slog.Debug("New backup initialized", "ID", backup.ID, "Name", backup.Name, "Status", backup.Status)
	return backup
}

// requestHomeAssistantBackup calls Home Assistant to create a backup a full backup
func (s *Service) requestHomeAssistantBackup(name string) (string, error) {
	slug, err := s.hassio.BackupFull(name)
	if err != nil {
		return "", err
	}

	return slug, nil
}

// processAndUploadBackup updates the backup with information from Home Assistant and uploads it to the remote drive
func (s *Service) processAndUploadBackup(backup *Backup) error {
	haBackup, err := s.hassio.GetBackup(backup.Slug)
	if err != nil {
		return err
	}

	s.updateHABackupDetails(backup, haBackup)

	key, err := s.uploadBackup(backup)
	if err != nil {
		return err
	}

	// Ensure backup.Drive is updated with the new upload detail
	backup.S3 = &s3.Object{
		Key: key,
	}

	return nil
}

// findBackupToDelete returns backup and index by ID
func (s *Service) findBackupToDelete(id string) (*Backup, int) {
	var backupToDelete *Backup
	deleteIndex := -1

	for i, backup := range s.backups {
		if backup.ID == id {
			backupToDelete = backup
			deleteIndex = i
			break
		}
	}

	return backupToDelete, deleteIndex
}

// deleteBackupInHomeAssistant calls home assistant to delete a backup
func (s *Service) deleteBackupInHomeAssistant(backupToDelete *Backup) error {
	err := s.hassio.DeleteBackup(backupToDelete.Slug)
	if err != nil {
		return handleBackupError(s, "failed to delete backup in Home Assistant", backupToDelete, err)
	}
	return nil
}

// deleteBackupFromS3 deletes a backup from the S3
func (s *Service) deleteBackupFromS3(backup *Backup) error {
	slog.Info("Deleting backup from S3", "backup", backup)
	err := s.s3.RemoveObject(context.Background(), s.config.S3.Bucket, backup.S3.Key, minio.RemoveObjectOptions{})
	if err != nil {
		return handleBackupError(s, "failed to delete backup from S3", backup, err)
	}
	return nil
}

// syncBackups synchronizes the backups by performing the following steps:
func (s *Service) syncBackups() error {
	// Wait if there is an ongoing backup
	if len(s.ongoingBackups) > 0 {
		slog.Info("Skipping synchronization due to ongoing backup operations.")
		return nil
	}

	// Get lock
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create a map of backups for easy access
	backupMap := make(map[string]*Backup)
	for _, backup := range s.backups {
		backupMap[backup.Name] = backup

		// NIL out HA and Drive
		// This will delete the backup from the map if it's not found in HA or Drive during the sync
		// Unsure if this is wanted behavior but sticking to it for now
		backup.HA = nil
		backup.S3 = nil
	}

	// Mark backups for deletion if needed
	if err := s.markExcessBackupsForDeletion(); err != nil {
		return err
	}

	// Keep HA backups up to date
	if err := s.updateOrDeleteHABackup(backupMap); err != nil {
		return err
	}

	// Keep Drive backups up to date
	if err := s.updateOrDeleteBackupsInBackend(backupMap); err != nil {
		return err
	}

	// Delete backups from the addon after making sure ha and drive are up to date
	s.deleteBackupFromAddon()

	// Update statused and sync backups to drive if needed
	backupsInS3 := s.updateBackupStatuses()
	if s.backupsInS3 == 0 || (len(s.backups) > backupsInS3 && backupsInS3 < s.backupsInS3) {
		slog.Debug("Syncing backups to Drive")
		if err := s.ensureS3Backups(backupsInS3); err != nil {
			return err
		}
	}

	// Sort and save backups
	return s.sortAndSaveBackups()
}

// updateBackupStatuses sets the status for each backup and counts backups on Drive.
func (s *Service) updateBackupStatuses() int {
	backupsInS3 := 0

	for _, backup := range s.backups {
		backupInHA, backupInS3 := backup.HA != nil, backup.S3 != nil
		if backupInHA && backupInS3 {
			backup.UpdateStatus(StatusSynced)
			backupsInS3++
		} else if backupInHA {
			backup.UpdateStatus(StatusHAOnly)
		} else if backupInS3 {
			backup.UpdateStatus(StatusS3Only)
			backupsInS3++
		}
	}

	return backupsInS3
}

// ensureS3Backups syncs the required number of backups to the drive.
func (s *Service) ensureS3Backups(backupsInS3 int) error {
	var uploadCount int
	if s.backupsInS3 > 0 {
		uploadCount = s.backupsInS3 - backupsInS3
	}

	haOnlyBackups := []*Backup{}
	for _, backup := range s.backups {
		if backup.Status == StatusHAOnly {
			haOnlyBackups = append(haOnlyBackups, backup)
		}
	}

	if uploadCount > 0 && uploadCount > len(haOnlyBackups) {
		uploadCount = len(haOnlyBackups)
	}

	for i := 0; i < uploadCount; i++ {
		backup := haOnlyBackups[i]
		if err := s.syncBackupToDriveAndLog(backup); err != nil {
			return err
		}
	}

	return nil
}

// syncBackupToDriveAndLog encapsulates the sync logic with logging.
func (s *Service) syncBackupToDriveAndLog(backup *Backup) error {
	slog.Info("Syncing backup to S3", "name", backup.Name)
	if err := s.syncBackupToDrive(backup); err != nil {
		slog.Error("Error syncing backup to S3", "name", backup.Name, "error", err)
		return err
	}
	return nil
}

// addHABackupsToMap adds Home Assistant backups to the backup map if it doesn't find one by name
func (s *Service) updateOrDeleteHABackup(backupMap map[string]*Backup) error {
	haBackups, err := s.hassio.ListBackups()
	if err != nil {
		return err
	}

	if len(haBackups) == 0 {
		slog.Debug("No backups found in Home Assistant")
		return nil
	}

	upToDate := true
	for _, haBackup := range haBackups {
		if haBackup.Type == "partial" {
			continue // Skip partial backups
		}

		if _, exists := backupMap[haBackup.Name]; !exists {
			slog.Info("Found untracked backup in Home Assistant", "name", haBackup.Name)
			backup := s.initializeBackup(haBackup.Name)
			s.updateHABackupDetails(backup, haBackup)

			backupMap[haBackup.Name] = backup
			upToDate = false
		} else {
			if !backupMap[haBackup.Name].KeepInHA {
				if !backupMap[haBackup.Name].Pinned {
					if err := s.hassio.DeleteBackup(haBackup.Slug); err != nil {
						return err
					}

					backupMap[haBackup.Name].HA = nil

					slog.Info("Deleted backup from Home Assistant", "name", haBackup.Name)
				}
			} else {
				backupMap[haBackup.Name].HA = haBackup
			}
		}
	}

	if upToDate {
		slog.Debug("Home Assistant backups up to date, no action taken")
	}

	return nil
}

// addDriveBackupsToMap adds bacckups found on the S3 to the backup map if it doesn't find one by name
func (s *Service) updateOrDeleteBackupsInBackend(backupMap map[string]*Backup) error {
	s3Backups := []*s3.Object{}
	objectCh := s.s3.ListObjects(context.Background(), s.config.S3.Bucket, minio.ListObjectsOptions{})
	for object := range objectCh {
		if object.Err != nil {
			slog.Error("could not list objects in S3: %v", object.Err)
			return fmt.Errorf("could not list objects: %v", object.Err)
		}

		s3Backups = append(s3Backups, &s3.Object{
			Key:      object.Key,
			Size:     float64(object.Size) / (1024 * 1024), // convert bytes to MB
			Modified: object.LastModified,
		})
	}

	if len(s3Backups) == 0 {
		slog.Debug("no backups found in S3")
		return nil
	}

	noUpdateNeeded := true
	for _, s3Backup := range s3Backups {
		name := strings.TrimSuffix(s3Backup.Key, ".tar")

		if _, exists := backupMap[name]; !exists {
			slog.Info("found untracked backup in S3", "name", s3Backup.Key)
			backup := s.initializeBackup(strings.TrimSuffix(s3Backup.Key, ".tar"))

			backup.S3 = s3Backup
			backup.Date = s3Backup.Modified
			backup.Size = s3Backup.Size

			noUpdateNeeded = false
		} else {
			if !backupMap[name].KeepInS3 {
				if !backupMap[name].Pinned {
					if err := s.s3.RemoveObject(context.Background(), s.config.S3.Bucket, s3Backup.Key, minio.RemoveObjectOptions{}); err != nil {
						return err
					}

					backupMap[name].S3 = nil
					slog.Info("Deleted backup from S3", "name", s3Backup.Key)
				}
			} else {
				backupMap[name].S3 = s3Backup
			}
		}
	}

	if noUpdateNeeded {
		slog.Debug("Backups in S3 up to date, no actions taken")
	}

	return nil
}

// deleteBackupFromAddon deletes a backup from the addon if it's not marked to be kept
func (s *Service) deleteBackupFromAddon() error {
	backupsToKeep := []*Backup{}
	for _, backup := range s.backups {
		if backup.HA != nil || backup.S3 != nil {
			if backup.KeepInHA || backup.KeepInS3 {
				backupsToKeep = append(backupsToKeep, backup)
			}
		}
	}

	s.backups = backupsToKeep

	return nil
}

// sortAndSaveBackups sorts the backup array by date, latest first and saves the state to file
func (s *Service) sortAndSaveBackups() error {
	sort.Slice(s.backups, func(i, j int) bool {
		return s.backups[i].Date.After(s.backups[j].Date)
	})

	if err := s.saveBackupsToFile(); err != nil {
		slog.Error("Error saving backup state after backup operation", "error", err)
		return err
	}

	return nil
}

// syncBackupToDrive uploads a backup to the drive if needed
func (s *Service) syncBackupToDrive(backup *Backup) error {
	if backup.S3 != nil {
		_, err := s.s3.StatObject(context.Background(), s.config.S3.Bucket, backup.Name, minio.StatObjectOptions{})
		if err == nil {
			return nil
		}
	}

	slog.Info("Syncing backup to S3", "name", backup.Name)
	backup.UpdateStatus(StatusSyncing)
	key, err := s.uploadBackup(backup)
	if err != nil {
		backup.UpdateStatus(StatusFailed)

		if err := s.saveBackupsToFile(); err != nil {
			slog.Error("Error saving backup state after backup operation", "error", err)
			return err
		}

		return handleBackupError(s, "failed to sync backup", backup, err)
	}

	// Ensure backup.Drive is updated with the new upload details
	backup.S3 = &s3.Object{
		Key: key,
	}
	backup.KeepInS3 = true

	backup.UpdateStatus(StatusSynced)
	return nil
}

// updateHABackupDetails updates the backup with information from HA
func (s *Service) updateHABackupDetails(backup *Backup, haBackup *hassio.Backup) {
	backup.HA = haBackup
	backup.Slug = haBackup.Slug
	backup.Date = haBackup.Date
	backup.Size = haBackup.Size
}

// updateS3BackupDetails updates the backup with information from S3
func (s *Service) updateS3BackupDetails(backup *Backup, s3Backup *s3.Object) error {
	slog.Info("Fetching backup attributes from S3", "name", backup.Name)

	objectName := backup.Name + ".tar"
	object, err := s.s3.StatObject(context.Background(), s.config.S3.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return fmt.Errorf("could not open object: %v", err)
	}

	attributes := &s3.Object{
		Key:      objectName,
		Size:     float64(object.Size) / (1024 * 1024), // convert bytes to MB
		Modified: object.LastModified,
	}

	slog.Info("Attributes fetched", "key", s3Backup.Key, "size", attributes.Size, "modified", attributes.Modified)

	backup.S3 = attributes
	backup.Date = attributes.Modified
	backup.Size = attributes.Size

	return nil
}

// markExcessBackupsForDeletion marks the oldest excess backups for deletion based on the given limit
func (s *Service) markExcessBackupsForDeletion() error {
	nonPinnedBackups := []*Backup{}
	for _, backup := range s.backups {
		if !backup.Pinned {
			nonPinnedBackups = append(nonPinnedBackups, backup)
		}
	}
	sort.Slice(nonPinnedBackups, func(i, j int) bool {
		return nonPinnedBackups[i].Date.Before(nonPinnedBackups[j].Date)
	})

	if s.config.BackupsInHA > 0 {
		if err := s.markForDeletion(nonPinnedBackups, s.config.BackupsInHA, true); err != nil {
			return err
		}
	} else {
		slog.Debug("Skipping deletion for Home Assistant backups; limit is set to 0.")
	}

	// Execute marking for Drive backups if a limit is set.
	if s.config.BackupsInS3 > 0 {
		if err := s.markForDeletion(nonPinnedBackups, s.config.BackupsInS3, false); err != nil {
			return err
		}
	} else {
		slog.Debug("Skipping deletion for Drive backups; limit is set to 0.")
	}

	return nil
}

// markForDeletion marks the oldest excess backups for deletion based on the given limit.
// The updateHA argument specifies whether to update KeepInHA or KeepOnDrive.
func (s *Service) markForDeletion(backups []*Backup, limit int, updateHA bool) error {
	excessCount := len(backups) - limit
	if excessCount <= 0 {
		return nil
	}

	for _, backup := range backups[:excessCount] {
		if updateHA {
			backup.KeepInHA = false
			slog.Debug("Marking backup for deletion in Home Assistant", "name", backup.Name)
		} else {
			backup.KeepInS3 = false
			slog.Debug("Marking backup for deletion on Drive", "name", backup.Name)
		}
	}
	return nil
}

// startBackupScheduler starts a go routine that will perform backups on a timer
func (s *Service) startBackupScheduler() {
	s.resetTimerForNextBackup()

	go func() {
		for {
			select {
			case <-s.backupTimer.C:
				slog.Info("Performing scheduled backup")

				if err := s.PerformBackup(""); err != nil {
					slog.Error("Error performing scheduled backup", "error", err)
				}
			case <-s.stopBackupChan:
				slog.Info("Stopping backup scheduler")
				return
			}
		}
	}()
}

// startBackupSyncScheduler starts a go routine that will perform syncs on a timer
func (s *Service) startBackupSyncScheduler() {
	s.syncTicker = time.NewTicker(s.syncInterval)

	go func() {
		for {
			select {
			case <-s.syncTicker.C:
				slog.Debug("Performing scheduled backup sync")

				if err := s.syncBackups(); err != nil {
					slog.Error("Error performing backup sync", "error", err)
				}
			case <-s.stopSyncChan:
				slog.Info("Stopping sync scheduler")
				s.syncTicker.Stop()
				return
			}
		}
	}()
}

// calculateDurationUntilNextBackup calculates the duration until the next backup should occur.
func (s *Service) calculateDurationUntilNextBackup() time.Duration {
	latestBackup := s.getLatestBackup()
	if latestBackup == nil {
		return 1 * time.Second
	}

	elapsed := time.Since(latestBackup.Date)
	if elapsed >= s.backupInterval {
		return 1 * time.Second
	}

	return s.backupInterval - elapsed
}

// resetTimerForNextBackup sets the timer for the next backup
func (s *Service) resetTimerForNextBackup() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.nextBackupIn = s.calculateDurationUntilNextBackup()
	s.nextBackupCalculatedAt = time.Now()

	if !s.backupTimer.Stop() && s.backupTimer != nil {
		select {
		case <-s.backupTimer.C: // Drain the channel
		default:
		}
	}

	slog.Info("Next backup scheduled", "timeLeft", s.nextBackupIn.String())
	s.backupTimer.Reset(s.nextBackupIn)
}

// listenForConfigChanges listen for changes to certain config values and takes action when the config changes
func (s *Service) listenForConfigChanges(configChan <-chan *config.Options) {
	for range configChan {
		newInterval := s.configService.GetBackupInterval()
		newBackupNameFormat := s.configService.GetBackupNameFormat()
		newBackupsInHA := s.configService.GetBackupsInHA()
		newBackupsInS3 := s.configService.GetBackupsInS3()

		if newInterval != s.backupInterval {
			s.backupInterval = newInterval
			s.resetTimerForNextBackup()
			slog.Info("Backup configuration updated", "new backupInterval", newInterval.String())
		}

		if newBackupsInHA != s.backupsInHA || newBackupsInS3 != s.backupsInS3 {
			s.backupsInHA = newBackupsInHA
			s.backupsInS3 = newBackupsInS3
			s.syncBackups()
		}

		if newBackupNameFormat != s.config.BackupNameFormat {
			slog.Info("Backup configuration updated", "new backupNameFormat", newBackupNameFormat)
		}

		if newBackupsInHA != s.backupsInHA {
			slog.Info("Backup configuration updated", "new backupsInHA", newBackupsInHA)
		}

		if newBackupsInS3 != s.backupsInS3 {
			slog.Info("Backup configuration updated", "new backupsOnDrive", newBackupsInS3)
		}
	}
}

func (s *Service) ResetBackups() error {
	file, err := os.Create("/data/backups.json")
	if err != nil {
		return err
	}

	defer file.Close()

	s.backups = []*Backup{}

	s.syncBackups()

	return nil
}

// loadBackupsFromFile populates the addons initial list of backups from a file on disk
func (s *Service) loadBackupsFromFile() {
	path := fmt.Sprintf("%s/backups.json", s.config.DataDirectory)
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("Error loading backups from file", "error", err)
		return
	}

	if err := json.Unmarshal(data, &s.backups); err != nil {
		slog.Error("Error unmarshaling backups:", "error", err)
	}
}

// saveBackupsToFile persists the addons list of backups to a file on disk
func (s *Service) saveBackupsToFile() error {
	data, err := json.Marshal(s.backups)
	if err != nil {
		return err
	}

	err = os.WriteFile(fmt.Sprintf("%s/%s", s.config.DataDirectory, "backups.json"), data, 0644)
	if err != nil {
		return err
	}

	return nil
}

// getLatestBackup returns the latest backup
func (s *Service) getLatestBackup() *Backup {
	var latestBackup *Backup

	for _, backup := range s.backups {
		if latestBackup == nil || backup.Date.After(latestBackup.Date) {
			latestBackup = backup
		}
	}

	return latestBackup
}

// generateBackupID return a backupID created from the current time
func (s *Service) generateBackupID() string {
	timestamp := time.Now().Format("20060102150405.000000000")
	return timestamp
}

// generateBackupName generates a backup name
func (s *Service) generateBackupName(requestName string) string {
	if requestName != "" {
		return requestName
	}

	format := s.config.BackupNameFormat
	now := time.Now().In(s.config.Timezone)
	format = strings.ReplaceAll(format, "{year}", now.Format("2006"))
	format = strings.ReplaceAll(format, "{month}", now.Format("01"))
	format = strings.ReplaceAll(format, "{day}", now.Format("02"))
	format = strings.ReplaceAll(format, "{hr24}", now.Format("15"))
	format = strings.ReplaceAll(format, "{min}", now.Format("04"))
	format = strings.ReplaceAll(format, "{sec}", now.Format("05"))

	return format
}

// removeOngoingBackup remeves the backup with provided ID from the list of ongoing bacups
func (s *Service) removeOngoingBackup(backupID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.ongoingBackups, backupID)
}

// uploadBackup uploads a backup from home assistant to the remote drive
func (s *Service) uploadBackup(backup *Backup) (string, error) {
	ctx := context.Background()
	contentType := "application/octet-stream"
	objectName := fmt.Sprintf("%s.%s", backup.Name, "tar")
	path := fmt.Sprintf("%s/%s.%s", "/backup", backup.HA.Slug, "tar")

	backup.UpdateStatus(StatusSyncing)
	info, err := s.s3.FPutObject(ctx, s.config.S3.Bucket, objectName, path, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		backup.UpdateStatus(StatusFailed)
		slog.Error("Error uploading backup to S3", err)
		return "", fmt.Errorf("could not upload object: %v", err)
	}

	return info.Key, nil
}

// handleBackupError takes a standard set of actions when a backup error occurs
func handleBackupError(s *Service, errMsg string, backup *Backup, err error) error {
	slog.Error(errMsg, err)
	if backup != nil {
		backup.UpdateStatus(StatusFailed)
		s.saveBackupsToFile() // Best effort to save state
	}
	return fmt.Errorf("%s: %v", errMsg, err)
}
