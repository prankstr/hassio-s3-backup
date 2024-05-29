package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"hassio-proton-drive-backup/models"
	"hassio-proton-drive-backup/pkg/clients"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// BackupStatus is a custom type to represent the status of the backup
type BackupStatus string

const (
	StatusDeleting    BackupStatus = "DELETING"    // Backup in being deleted
	StatusPending     BackupStatus = "PENDING"     // Backup is initalized but no action taken
	StatusRunning     BackupStatus = "RUNNING"     // Backup is being created in Home Assistant
	StatusSynced      BackupStatus = "SYNCED"      // Backup is present in both Home Assistant and drive
	StatusHAOnly      BackupStatus = "HAONLY"      // Backup is only present in  Home Assistant
	StatusStorageOnly BackupStatus = "STORAGEONLY" // Backup is only present on drive
	StatusSyncing     BackupStatus = "SYNCING"     // Backups is being uploaded to drive
	StatusFailed      BackupStatus = "FAILED"      // Backup process failed somewhere
)

// Backup represents the details and status of a backup process
type Backup struct {
	Date          time.Time      `json:"date"`
	Storage       *DirectoryItem `json:"storage"`
	HA            *HassBackup    `json:"ha"`
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Status        BackupStatus   `json:"status"`
	Slug          string         `json:"slug"`
	ErrorMessage  string         `json:"errorMessage"`
	Size          float64        `json:"size"`
	KeepInHA      bool           `json:"keepInHA"`
	KeepInStorage bool           `json:"keepInStorage"`
	Pinned        bool           `json:"pinned"`
}

// HAData is a selection of metadata from the HA backups
type HAData struct {
	Date time.Time `json:"date"`
	Slug string    `json:"slug"`
	Size float64   `json:"size"`
}

type BackupRequest struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProtonLink string `json:"protonLink"` // Proton Drive Link to file
	Slug       string `json:"slug"`       // Backup slug from Home assistant
}

// UpdateStatus updates the status of the backup
func (b *Backup) UpdateStatus(status BackupStatus) {
	b.Status = status
}

type BackupService struct {
	nextBackupCalculatedAt time.Time
	storageService         models.StorageService
	stopBackupChan         chan struct{}
	configService          *ConfigService
	config                 *models.Config
	syncTicker             *time.Ticker
	ongoingBackups         map[string]struct{}
	stopSyncChan           chan struct{}
	backupTimer            *time.Timer
	hassioApi              clients.HassioApiClient
	storageProvider        string
	backups                []*models.Backup
	backupsInStorage       int
	backupsInHA            int
	nextBackupIn           time.Duration
	backupInterval         time.Duration
	syncInterval           time.Duration
	mutex                  sync.Mutex
}

func NewBackupService(hassioApiClient clients.HassioApiClient, storage models.StorageService, configService *ConfigService) *BackupService {
	config := configService.config

	service := &BackupService{
		hassioApi:       hassioApiClient,
		storageService:  storage,
		storageProvider: "Proton Drive",
		configService:   configService,
		config:          config,

		stopBackupChan:   make(chan struct{}),
		stopSyncChan:     make(chan struct{}),
		backupInterval:   configService.GetBackupInterval(),
		backupsInHA:      configService.GetBackupsInHA(),
		backupsInStorage: configService.GetBackupsInStorage(),
		syncInterval:     1 * time.Minute, // set the interval for sync

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
	go service.listenForConfigChanges(configService.configChangeChan)

	return service
}

// PerformBackup creates a new backup and uploads it to the remote drive
func (s *BackupService) PerformBackup(name string) error {
	backup := s.initializeBackup(name)

	// Track ongoing backups to avoid syncing or any other manipulation in the meantime
	s.ongoingBackups[backup.ID] = struct{}{}

	slog.Info("Backup started", "name", backup.Name)
	slog.Info("Requesting backup from Home Assistant", "name", backup.Name)

	// Create backup in Home Assistant
	backup.UpdateStatus(models.StatusRunning)
	slug, err := s.requestHomeAssistantBackup(backup.Name)
	if err != nil {
		slog.Error("Failed to request backup from Home Assistant", "BackupName", backup.Name, "Error", err)
		backup.UpdateStatus(models.StatusFailed)
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
		slog.Error("Error syncing backup to storage backend", "name", backup.Name, "error", err)
		backup.UpdateStatus(models.StatusFailed)
		s.removeOngoingBackup(backup.ID)
		return err
	}

	// Remove ongoing backup and save state
	backup.UpdateStatus(models.StatusSynced)
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
func (s *BackupService) DeleteBackup(id string) error {
	backupToDelete, deleteIndex := s.findBackupToDelete(id)
	if backupToDelete == nil {
		slog.Error("Backup not found for deletion", "id", id)
		return nil // or return an error indicating that the backup was not found
	}

	slog.Info("Initiating backup deletion", "Backup", backupToDelete.Name, "BackupID", backupToDelete.ID)
	backupToDelete.UpdateStatus(models.StatusDeleting)

	// Delete backup from Home Assistant
	slog.Debug("Deleting backup from Home Assistant", "ID", backupToDelete.ID)
	if err := s.deleteBackupInHomeAssistant(backupToDelete); err != nil {
		slog.Error("Failed to delete backup in Home Assistant", "Backup", backupToDelete.Name, "error", err)
	} else {
		slog.Info("Backup deleted from Home Assistant", "name", backupToDelete.Name)
	}

	// Delete backup from the drive
	if backupToDelete.Storage != nil {
		slog.Debug("Deleting backup from storage backend", "ID", backupToDelete.ID)
		if err := s.deleteBackupFromDrive(backupToDelete); err != nil {
			slog.Error("Failed to delete backup from storage backend", "Backup", backupToDelete.Name, "Error", err)
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
func (s *BackupService) RestoreBackup(id string) error {
	var backupToRestore *models.Backup

	for _, backup := range s.backups {
		if backup.ID == id {
			backupToRestore = backup
			break
		}
	}

	slog.Info("Attempting to restore to backup", "name", backupToRestore.Name)
	err := s.hassioApi.RestoreBackup(backupToRestore.HA.Slug)
	if err != nil {
		return fmt.Errorf("failed to restore backup in Home Assistant: %v", err)
	}

	slog.Info("Restored to backup", "name", backupToRestore.Name)
	return nil
}

// DownloadBackup downloads a backup to the specified directory
// Note: might not be needed, download can be done manually from drive and then uploaded to Home Assistant
func (s *BackupService) DownloadBackup(id string) error {
	var backup *models.Backup

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
	backup.UpdateStatus(models.StatusSyncing)

	// Download the backup file
	reader, err := s.storageService.DownloadBackup(backup.Storage.Identifier)
	if err != nil {
		slog.Error("Failed to download backup", "backup", backup.Name, "error", err)
		backup.UpdateStatus(models.StatusStorageOnly)
		return err
	}
	defer reader.Close()

	// Write the backup file to disk
	filePath := filepath.Join("/backup", backup.Name+".tar")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open file for writing", "backup", backup.Name, "filePath", filePath, "error", err)
		backup.UpdateStatus(models.StatusStorageOnly)
		return err
	}
	defer file.Close()

	// Copy the downloaded file to disk
	if _, err := io.Copy(file, reader); err != nil {
		slog.Error("Failed to write backup to disk", "backup", backup.Name, "filePath", filePath, "error", err)
		backup.UpdateStatus(models.StatusStorageOnly)
		return err
	}

	slog.Info("Backup downloaded successfully", "backup", backup.Name, "filePath", filePath)
	backup.KeepInHA = true
	backup.UpdateStatus(models.StatusSynced)

	return nil
}

// PinBackup pins a backup to prevent it from being deleted
func (s *BackupService) PinBackup(id string) error {
	for _, backup := range s.backups {
		if backup.ID == id {
			backup.KeepInHA = true
			backup.KeepInStorage = true
			backup.Pinned = true
			slog.Info("Backup pinned", "name", backup.Name)
			return s.saveBackupsToFile()
		}
	}

	return errors.New("backup not found")
}

// UninBackup unpins a backup to allow it to be deleted
func (s *BackupService) UnpinBackup(id string) error {
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
func (s *BackupService) ListBackups() []*models.Backup {
	return s.backups
}

// TimeUntilNextBackup returns the time until next backup in milliseconds
func (s *BackupService) TimeUntilNextBackup() int64 {
	return time.Until(s.nextBackupCalculatedAt.Add(s.nextBackupIn)).Milliseconds()
}

// NameExists checks if a backup with the given name exists
func (s *BackupService) NameExists(name string) bool {
	generatedName := s.generateBackupName(name)

	for _, backup := range s.backups {
		if backup.Name == generatedName {
			return true
		}
	}

	return false
}

// initializeBackup return a new internal backup object
func (s *BackupService) initializeBackup(name string) *models.Backup {
	backup := &models.Backup{
		ID:            s.generateBackupID(),
		Name:          s.generateBackupName(name),
		Date:          time.Now().In(s.config.Timezone),
		Status:        models.StatusPending,
		KeepInHA:      true,
		KeepInStorage: true,
		Storage:       nil,
		HA:            nil,
	}

	s.backups = append([]*models.Backup{backup}, s.backups...)

	slog.Debug("New backup initialized", "ID", backup.ID, "Name", backup.Name, "Status", backup.Status)
	return backup
}

// requestHomeAssistantBackup calls Home Assistant to create a backup a full backup
func (s *BackupService) requestHomeAssistantBackup(name string) (string, error) {
	slug, err := s.hassioApi.BackupFull(name)
	if err != nil {
		return "", err
	}

	return slug, nil
}

// processAndUploadBackup updates the backup with information from Home Assistant and uploads it to the remote drive
func (s *BackupService) processAndUploadBackup(backup *models.Backup) error {
	haBackup, err := s.hassioApi.GetBackup(backup.Slug)
	if err != nil {
		return err
	}

	s.updateBackupDetailsFromHA(backup, haBackup)

	id, err := s.uploadBackup(backup)
	if err != nil {
		return err
	}

	// Ensure backup.Drive is updated with the new upload detail
	backup.Storage = &models.DirectoryItem{
		Identifier: id,
	}

	return nil
}

// findBackupToDelete returns backup and index by ID
func (s *BackupService) findBackupToDelete(id string) (*models.Backup, int) {
	var backupToDelete *models.Backup
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
func (s *BackupService) deleteBackupInHomeAssistant(backupToDelete *models.Backup) error {
	err := s.hassioApi.DeleteBackup(backupToDelete.Slug)
	if err != nil {
		return handleBackupError(s, "failed to delete backup in Home Assistant", backupToDelete, err)
	}
	return nil
}

// deleteBackupFromDrived deletes a backup from the remote drive
func (s *BackupService) deleteBackupFromDrive(backupToDelete *models.Backup) error {
	err := s.storageService.DeleteBackup(backupToDelete.Storage.Identifier)
	if err != nil {
		return handleBackupError(s, "failed to delete backup from Drive", backupToDelete, err)
	}
	return nil
}

// syncBackups synchronizes the backups by performing the following steps:
func (s *BackupService) syncBackups() error {
	// Wait if there is an ongoing backup
	if len(s.ongoingBackups) > 0 {
		slog.Info("Skipping synchronization due to ongoing backup operations.")
		return nil
	}

	// Get lock
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create a map of backups for easy access
	backupMap := make(map[string]*models.Backup)
	for _, backup := range s.backups {
		backupMap[backup.Name] = backup

		// NIL out HA and Drive
		// This will delete the backup from the map if it's not found in HA or Drive during the sync
		// Unsure if this is wanted behavior but sticking to it for now
		backup.HA = nil
		backup.Storage = nil
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
	backupsOnDrive := s.updateBackupStatuses()
	if s.backupsInStorage == 0 || (len(s.backups) > backupsOnDrive && backupsOnDrive < s.backupsInStorage) {
		slog.Debug("Syncing backups to Drive")
		if err := s.syncNeededBackupsToDrive(backupsOnDrive); err != nil {
			return err
		}
	}

	// Sort and save backups
	return s.sortAndSaveBackups()
}

// updateBackupStatuses sets the status for each backup and counts backups on Drive.
func (s *BackupService) updateBackupStatuses() int {
	backupsInStorage := 0

	for _, backup := range s.backups {
		backupInHA, backupInStorage := backup.HA != nil, backup.Storage != nil

		if backupInHA && backupInStorage {
			backup.UpdateStatus(models.StatusSynced)
			backupsInStorage++
		} else if backupInHA {
			backup.UpdateStatus(models.StatusHAOnly)
		} else if backupInStorage {
			backup.UpdateStatus(models.StatusStorageOnly)
			backupsInStorage++
		}
	}

	return backupsInStorage
}

// syncNeededBackupsToDrive syncs the required number of backups to the drive.
func (s *BackupService) syncNeededBackupsToDrive(backupsOnDrive int) error {
	var uploadCount int
	if s.backupsInStorage > 0 {
		uploadCount = s.backupsInStorage - backupsOnDrive
	}

	haOnlyBackups := []*models.Backup{}
	for _, backup := range s.backups {
		if backup.Status == models.StatusHAOnly {
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
func (s *BackupService) syncBackupToDriveAndLog(backup *models.Backup) error {
	slog.Info("Syncing backup to storage backend", "name", backup.Name)
	if err := s.syncBackupToDrive(backup); err != nil {
		slog.Error("Error syncing backup to storage backend", "name", backup.Name, "error", err)
		return err
	}
	return nil
}

// addHABackupsToMap adds Home Assistant backups to the backup map if it doesn't find one by name
func (s *BackupService) updateOrDeleteHABackup(backupMap map[string]*models.Backup) error {
	haBackups, err := s.hassioApi.ListBackups()
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
			s.updateBackupDetailsFromHA(backup, haBackup)

			backupMap[haBackup.Name] = backup
			upToDate = false
		} else {
			if !backupMap[haBackup.Name].KeepInHA {
				if !backupMap[haBackup.Name].Pinned {
					if err := s.hassioApi.DeleteBackup(haBackup.Slug); err != nil {
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

// addDriveBackupsToMap adds bacckups found on the storage backend to the backup map if it doesn't find one by name
func (s *BackupService) updateOrDeleteBackupsInBackend(backupMap map[string]*models.Backup) error {
	storageBackups, err := s.storageService.ListBackups()
	if err != nil {
		return err
	}

	fmt.Println(storageBackups)
	if len(storageBackups) == 0 {
		slog.Debug("No backups found on the storage backend")
		return nil
	}

	noUpdateNeeded := true
	for _, storageBackup := range storageBackups {
		if _, exists := backupMap[storageBackup.Name]; !exists {
			slog.Info("Found untracked backup in storage backend", "name", storageBackup.Name)
			backup := s.initializeBackup(storageBackup.Name)

			if err := s.updateBackupDetailsFromDrive(backup, storageBackup); err != nil {
				return err
			}

			noUpdateNeeded = false
		} else {
			if !backupMap[storageBackup.Name].KeepInStorage {
				if !backupMap[storageBackup.Name].Pinned {
					if err := s.storageService.DeleteBackup(storageBackup.Identifier); err != nil {
						return err
					}

					backupMap[storageBackup.Name].Storage = nil
					slog.Info("Deleted backup from storage backend", "name", storageBackup.Name)
				}
			} else {
				backupMap[storageBackup.Name].Storage = storageBackup
			}
		}
	}

	if noUpdateNeeded {
		slog.Debug("Backups in storage backend to date, no actions taken")
	}

	return nil
}

// deleteBackupFromAddon deletes a backup from the addon if it's not marked to be kept
func (s *BackupService) deleteBackupFromAddon() error {
	backupsToKeep := []*models.Backup{}
	for _, backup := range s.backups {
		if backup.HA != nil || backup.Storage != nil {
			if backup.KeepInHA || backup.KeepInStorage {
				backupsToKeep = append(backupsToKeep, backup)
			}
		}
	}

	s.backups = backupsToKeep

	return nil
}

// sortAndSaveBackups sorts the backup array by date, latest first and saves the state to file
func (s *BackupService) sortAndSaveBackups() error {
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
func (s *BackupService) syncBackupToDrive(backup *models.Backup) error {
	if backup.Storage != nil {
		exists := s.storageService.FileExists(backup.Storage.Identifier)
		if exists {
			return nil
		}
	}

	slog.Info("Syncing backup to storage backend", "name", backup.Name)
	backup.UpdateStatus(models.StatusSyncing)
	link, err := s.uploadBackup(backup)
	if err != nil {
		backup.UpdateStatus(models.StatusFailed)

		if err := s.saveBackupsToFile(); err != nil {
			slog.Error("Error saving backup state after backup operation", "error", err)
			return err
		}

		return handleBackupError(s, "failed to sync backup", backup, err)
	}

	// Ensure backup.Drive is updated with the new upload details
	backup.Storage = &models.DirectoryItem{
		Name:       backup.Name,
		Identifier: link,
	}
	backup.KeepInStorage = true

	backup.UpdateStatus(models.StatusSynced)
	return nil
}

// updateBackupDetailsFromHA updates the backup with information from HA
func (s *BackupService) updateBackupDetailsFromHA(backup *models.Backup, haBackup *models.HassBackup) {
	backup.HA = haBackup
	backup.Slug = haBackup.Slug
	backup.Date = haBackup.Date
	backup.Size = haBackup.Size
}

// updateBackupDetailsFromHA updates the backup with information from HA
func (s *BackupService) updateBackupDetailsFromDrive(backup *models.Backup, driveBackup *models.DirectoryItem) error {
	slog.Info("Fetching backup attributes from storage backend", "name", backup.Name)
	attributes, err := s.storageService.GetBackupAttributes(driveBackup.Identifier)
	if err != nil {
		return err
	}

	slog.Info("Attributes fetched", "name", driveBackup.Name, "size", attributes.Size, "modified", attributes.Modified)

	backup.Storage = driveBackup
	backup.Name = driveBackup.Name
	backup.Date = attributes.Modified
	backup.Size = attributes.Size

	return nil
}

// markExcessBackupsForDeletion marks the oldest excess backups for deletion based on the given limit
func (s *BackupService) markExcessBackupsForDeletion() error {
	nonPinnedBackups := []*models.Backup{}
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
	if s.config.BackupsInStorage > 0 {
		if err := s.markForDeletion(nonPinnedBackups, s.config.BackupsInStorage, false); err != nil {
			return err
		}
	} else {
		slog.Debug("Skipping deletion for Drive backups; limit is set to 0.")
	}

	return nil
}

// markForDeletion marks the oldest excess backups for deletion based on the given limit.
// The updateHA argument specifies whether to update KeepInHA or KeepOnDrive.
func (s *BackupService) markForDeletion(backups []*models.Backup, limit int, updateHA bool) error {
	excessCount := len(backups) - limit
	if excessCount <= 0 {
		return nil
	}

	for _, backup := range backups[:excessCount] {
		if updateHA {
			backup.KeepInHA = false
			slog.Debug("Marking backup for deletion in Home Assistant", "name", backup.Name)
		} else {
			backup.KeepInStorage = false
			slog.Debug("Marking backup for deletion on Drive", "name", backup.Name)
		}
	}
	return nil
}

// startBackupScheduler starts a go routine that will perform backups on a timer
func (s *BackupService) startBackupScheduler() {
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
func (s *BackupService) startBackupSyncScheduler() {
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
func (s *BackupService) calculateDurationUntilNextBackup() time.Duration {
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
func (s *BackupService) resetTimerForNextBackup() {
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
func (s *BackupService) listenForConfigChanges(configChan <-chan *models.Config) {
	for range configChan {
		newInterval := s.configService.GetBackupInterval()
		newBackupNameFormat := s.configService.GetBackupNameFormat()
		newBackupsInHA := s.configService.GetBackupsInHA()
		newBackupsInStorage := s.configService.GetBackupsInStorage()

		if newInterval != s.backupInterval {
			s.backupInterval = newInterval
			s.resetTimerForNextBackup()
			slog.Info("Backup configuration updated", "new backupInterval", newInterval.String())
		}

		if newBackupsInHA != s.backupsInHA || newBackupsInStorage != s.backupsInStorage {
			s.backupsInHA = newBackupsInHA
			s.backupsInStorage = newBackupsInStorage
			s.syncBackups()
		}

		if newBackupNameFormat != s.config.BackupNameFormat {
			slog.Info("Backup configuration updated", "new backupNameFormat", newBackupNameFormat)
		}

		if newBackupsInHA != s.backupsInHA {
			slog.Info("Backup configuration updated", "new backupsInHA", newBackupsInHA)
		}

		if newBackupsInStorage != s.backupsInStorage {
			slog.Info("Backup configuration updated", "new backupsOnDrive", newBackupsInStorage)
		}
	}
}

func (s *BackupService) ResetBackups() error {
	file, err := os.Create("/data/backups.json")
	if err != nil {
		return err
	}

	defer file.Close()

	s.backups = []*models.Backup{}

	s.syncBackups()

	return nil
}

// loadBackupsFromFile populates the addons initial list of backups from a file on disk
func (s *BackupService) loadBackupsFromFile() {
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
func (s *BackupService) saveBackupsToFile() error {
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
func (s *BackupService) getLatestBackup() *models.Backup {
	var latestBackup *models.Backup

	for _, backup := range s.backups {
		if latestBackup == nil || backup.Date.After(latestBackup.Date) {
			latestBackup = backup
		}
	}

	return latestBackup
}

// generateBackupID return a backupID created from the current time
func (s *BackupService) generateBackupID() string {
	timestamp := time.Now().Format("20060102150405.000000000")
	return timestamp
}

// generateBackupName generates a backup name
func (s *BackupService) generateBackupName(requestName string) string {
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
func (s *BackupService) removeOngoingBackup(backupID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.ongoingBackups, backupID)
}

// uploadBackup uploads a backup from home assistant to the remote drive
func (s *BackupService) uploadBackup(backup *models.Backup) (string, error) {
	path := fmt.Sprintf("%s/%s.%s", "/backup", backup.HA.Slug, "tar")

	backup.UpdateStatus(models.StatusSyncing)
	id, err := s.storageService.UploadBackup(fmt.Sprintf("%s.%s", backup.Name, "tar"), path)
	if err != nil {
		backup.UpdateStatus(models.StatusFailed)
		slog.Error("Error uploading backup to storage backend", err)
		return "", err
	}

	return id, nil
}

// handleBackupError takes a standard set of actions when a backup error occurs
func handleBackupError(s *BackupService, errMsg string, backup *models.Backup, err error) error {
	slog.Error(errMsg, err)
	if backup != nil {
		backup.UpdateStatus(models.StatusFailed)
		s.saveBackupsToFile() // Best effort to save state
	}
	return fmt.Errorf("%s: %v", errMsg, err)
}
