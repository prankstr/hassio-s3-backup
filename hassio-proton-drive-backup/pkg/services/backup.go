package services

import (
	"encoding/json"
	"fmt"
	"hassio-proton-drive-backup/models"
	"hassio-proton-drive-backup/pkg/clients"
	"os"
	"sort"
	"sync"
	"time"

	"log/slog"
)

type BackupService struct {
	hassioApi     clients.HassioApiClient
	drive         models.Drive
	driveProvider string
	configService *ConfigService
	config        *models.Config

	syncTicker   *time.Ticker
	syncInterval time.Duration
	stopSyncChan chan struct{}

	backups                []*models.Backup
	backupTimer            *time.Timer
	backupInterval         time.Duration
	backupsToKeep          int
	stopBackupChan         chan struct{}
	nextBackupIn           time.Duration
	nextBackupCalculatedAt time.Time
	ongoingBackups         map[string]struct{}

	mutex sync.Mutex
}

func NewBackupService(hassioApiClient clients.HassioApiClient, drive models.Drive, configService *ConfigService) *BackupService {
	config := configService.config

	service := &BackupService{
		hassioApi:     hassioApiClient,
		drive:         drive,
		driveProvider: "Proton Drive",
		configService: configService,
		config:        config,

		stopBackupChan: make(chan struct{}),
		stopSyncChan:   make(chan struct{}),
		backupInterval: configService.GetBackupInterval(),
		backupsToKeep:  configService.GetBackupsToKeep(),
		syncInterval:   1 * time.Minute, // set the interval for sync

		ongoingBackups: make(map[string]struct{}),
	}

	// Initial load and sync, run sync in routine to not block startup
	service.loadBackupsFromFile()
	service.syncBackups()

	// Initialize backupTimer with a dummy duration, it will be reset later
	service.backupTimer = time.NewTimer(time.Hour)
	service.backupTimer.Stop() // Stop the dummy timer immediately

	// Start scheduled backups and syncs
	go service.startBackupScheduler()
	go service.startBackupSyncScheduler()
	go service.listenForConfigChanges(configService.configChangeChan)

	return service
}

// PerformBackup creates a new backup and uploads it to the remote drive
func (s *BackupService) PerformBackup(name string) error {
	backup := s.initializeBackup(name)
	s.ongoingBackups[backup.ID] = struct{}{}

	slog.Info("Backup started", "BackupName", backup.Name)
	slog.Debug("Requesting backup from Home Assistant", "BackupName", backup.Name)

	backup.UpdateStatus(models.StatusRunning)
	slug, err := s.requestHomeAssistantBackup(backup)
	if err != nil {
		slog.Error("Failed to request backup from Home Assistant", "BackupName", backup.Name, "Error", err)
		backup.UpdateStatus(models.StatusFailed)
		s.removeOngoingBackup(backup.ID)
		return err
	}
	backup.HA.Slug = slug

	if err := s.saveBackupsToFile(); err != nil {
		slog.Error("Error saving backup state after Home Assistant request", "name", backup.Name, "error", err)
		return err
	}

	if err := s.processAndUploadBackup(backup); err != nil {
		slog.Error(fmt.Sprintf("Error syncing backup with %s", s.driveProvider), "name", backup.Name, "error", err)
		backup.UpdateStatus(models.StatusFailed)
		s.removeOngoingBackup(backup.ID)
		return err
	}
	backup.UpdateStatus(models.StatusSynced)
	slog.Info("Backup process completed successfully", "BackupName", backup.Name, "Status", backup.Status)

	if err := s.saveBackupsToFile(); err != nil {
		slog.Error("Error saving backup state after syncing", "name", backup.Name, "error", err)
		return err
	}

	s.removeOngoingBackup(backup.ID)
	s.resetTimerForNextBackup()

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

	slog.Debug("Deleting backup from Home Assistant", "ID", backupToDelete.ID)
	if err := s.deleteBackupInHomeAssistant(backupToDelete); err != nil {
		slog.Error("Failed to delete backup in Home Assistant", "Backup", backupToDelete.Name, "error", err)
	} else {
		slog.Debug("Backup deleted from Home Assistant", "ID", backupToDelete.ID)
	}

	slog.Debug(fmt.Sprintf("Deleting backup from %s", s.driveProvider), "ID", backupToDelete.ID)
	if err := s.deleteBackupFromDrive(backupToDelete); err != nil {
		slog.Error(fmt.Sprintf("Failed to delete backup from %s", s.driveProvider), "Backup", backupToDelete.Name, "Error", err)
	}

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

	slog.Debug("Backup deletion process completed", "name", backupToDelete.Name)
	return nil
}

// RestoreBackup calls home assistant to restore a backup
func (s *BackupService) RestoreBackup(slug string) error {
	var backupToRestore *models.Backup

	for _, backup := range s.backups {
		if backup.HA.Slug == slug {
			backupToRestore = backup
			break
		}
	}

	err := s.hassioApi.RestoreBackup(backupToRestore.HA.Slug)
	if err != nil {
		return fmt.Errorf("failed to restore backup in Home Assistant: %v", err)
	}

	slog.Info("Restored to backup", "name", backupToRestore.Name)
	return nil
}

// List Backups returns the addons list of backups
func (s *BackupService) ListBackups() []*models.Backup {
	return s.backups
}

// TimeUntilNextBackup returns the time until next backup in milliseconds
func (s *BackupService) TimeUntilNextBackup() int64 {
	return time.Until(s.nextBackupCalculatedAt.Add(s.nextBackupIn)).Milliseconds()
}

// initializeBackup return a new internal backup object
func (s *BackupService) initializeBackup(name string) *models.Backup {
	backup := &models.Backup{
		ID:     s.generateBackupID(),
		Name:   s.generateBackupName(name),
		Date:   time.Now().In(s.config.Timezone),
		Status: models.StatusPending,
		Drive:  &models.DirectoryData{},
		HA:     &models.HassBackup{},
	}

	s.backups = append([]*models.Backup{backup}, s.backups...)

	slog.Debug("New backup initialized", "ID", backup.ID, "Name", backup.Name, "Status", backup.Status)
	return backup
}

// requestHomeAssistantBackup call home assistant to create a backup of home assistant
func (s *BackupService) requestHomeAssistantBackup(backup *models.Backup) (string, error) {
	slog.Info("Requesting backup from Home Assistant", "name", backup.Name)
	slug, err := s.hassioApi.BackupFull(backup.Name)
	if err != nil {
		return "", err
	}

	return slug, nil
}

// processAndUploadBackup updates the backup with information from Home Assistant and uploads it to the remote drive
func (s *BackupService) processAndUploadBackup(backup *models.Backup) error {
	haBackup, err := s.hassioApi.GetBackup(backup.HA.Slug)
	if err != nil {
		return err
	}

	s.updateBackupDetailsFromHA(backup, haBackup)

	link, err := s.uploadBackup(backup)
	if err != nil {
		return err
	}

	backup.Drive.Identifier = link

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
	err := s.hassioApi.DeleteBackup(backupToDelete.HA.Slug)
	if err != nil {
		return handleBackupError(s, "failed to delete backup in Home Assistant", backupToDelete, err)
	}
	return nil
}

// deleteBackupFromDrived deletes a backup from the remote drive
func (s *BackupService) deleteBackupFromDrive(backupToDelete *models.Backup) error {
	err := s.drive.DeleteFileByID(backupToDelete.Drive.Identifier)
	if err != nil {
		return handleBackupError(s, "failed to delete backup from Drive", backupToDelete, err)
	}
	return nil
}

// syncBackups makes sure the wanted state between home assistant and the remove drive is upheld
func (s *BackupService) syncBackups() error {
	if len(s.ongoingBackups) > 0 {
		slog.Info("Skipping synchronization due to ongoing backup")
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Initialize UnifiedBackups with local backups
	backupMap := make(map[string]*models.Backup)
	for _, backup := range s.backups {
		backup.MarkedForDeletion = true
		backupMap[backup.Name] = backup
	}

	// Fetch HA backups
	if err := s.addHABackupsToMap(backupMap); err != nil {
		return err
	}

	// Fetch Drive backups
	if err := s.addDriveBackupsToMap(backupMap); err != nil {
		return err
	}

	filteredBackups := []*models.Backup{}
	for _, backup := range s.backups {
		if !backup.MarkedForDeletion {
			filteredBackups = append(filteredBackups, backup)
		} else {
			slog.Info("Deleting local(addon) backup", "name", backup.Name, "ID", backup.ID)
		}
	}
	s.backups = filteredBackups

	// Function to set status to HAONLY, DRIVEONLY or SYNCED
	for _, backup := range s.backups {
		if backup.HA.Slug == "" {
			backup.UpdateStatus(models.StatusDriveOnly)
		} else if backup.Drive.Identifier == "" {
			backup.UpdateStatus(models.StatusHAOnly)
		} else {
			backup.UpdateStatus(models.StatusSynced)
		}
	}

	// Sync to Drive
	for _, backup := range s.backups {
		if backup.Status == models.StatusHAOnly {
			if err := s.syncBackupToDrive(backup); err != nil {
				slog.Error(fmt.Sprintf("Error syncing backup to %s", s.driveProvider), "name", backup.Name, "error", err)
				return err
			}
		}
	}

	return s.sortAndSaveBackups()
}

// addHABackupsToMap adds Home Assistant backups to the backup map if it doesn't find one by name
func (s *BackupService) addHABackupsToMap(backupMap map[string]*models.Backup) error {
	haBackups, err := s.hassioApi.ListBackups()
	if err != nil {
		return err
	}

	if len(haBackups) == 0 {
		slog.Debug("No backups found in Home Assistant")
		return nil
	}

	noUpdateNeeded := true
	for _, haBackup := range haBackups {
		if _, exists := backupMap[haBackup.Name]; !exists {
			slog.Debug("Initializing backup found in Home Assistant", "name", haBackup.Name)
			backup := s.initializeBackup(haBackup.Name)

			s.updateBackupDetailsFromHA(backup, haBackup)
			noUpdateNeeded = false
		} else {
			backupMap[haBackup.Name].MarkedForDeletion = false

			// Don't really have to do this but might as well..
			backupMap[haBackup.Name].HA = haBackup
		}
	}

	if noUpdateNeeded {
		slog.Debug("No updates needed for backups from Home Assistant")
	}

	return nil
}

// addDriveBackupsToMap adds Drive backups to the backup map if it doesn't find one by name
func (s *BackupService) addDriveBackupsToMap(backupMap map[string]*models.Backup) error {
	driveBackups, err := s.drive.ListBackupDirectory()
	if err != nil {
		return err
	}

	if len(driveBackups) == 0 {
		slog.Debug(fmt.Sprintf("No backups found in %s", s.driveProvider))
		return nil
	}

	noUpdateNeeded := true
	for _, driveBackup := range driveBackups {
		if _, exists := backupMap[driveBackup.Name]; !exists {
			slog.Debug(fmt.Sprintf("Initializing backup found in %s", s.driveProvider), "name", driveBackup.Name)
			backup := s.initializeBackup(driveBackup.Name)

			if err := s.updateBackupDetailsFromDrive(backup, driveBackup); err != nil {
				return err
			}

			noUpdateNeeded = false
		} else {
			backupMap[driveBackup.Name].MarkedForDeletion = false

			// Don't really have to do this but might as well..
			backupMap[driveBackup.Name].Drive = driveBackup
		}
	}

	if noUpdateNeeded {
		slog.Debug(fmt.Sprintf("No updates needed for backups from %s", s.driveProvider))
	}

	return nil
}

// sortAndSaveBackups sorts the backup array by date, latest first
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
	if backup.Drive.Identifier == "" {
		slog.Info(fmt.Sprintf("Syncing backup to %s", s.driveProvider), "name", backup.Name)
	} else {
		exists := s.drive.FileExists(backup.Drive.Identifier)
		if !exists {
			slog.Info(fmt.Sprintf("Syncing backup to %s", s.driveProvider), "name", backup.Name)
		} else {
			return nil
		}
	}

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

	backup.Drive.Identifier = link
	backup.UpdateStatus(models.StatusSynced)
	return nil
}

// updateBackupDetailsFromHA updates the backup with information from HA
func (s *BackupService) updateBackupDetailsFromHA(backup *models.Backup, haBackup *models.HassBackup) {
	backup.HA.Slug = haBackup.Slug
	backup.HA.Type = haBackup.Type
	backup.Date = haBackup.Date
	backup.Size = haBackup.Size
}

// updateBackupDetailsFromHA updates the backup with information from HA
func (s *BackupService) updateBackupDetailsFromDrive(backup *models.Backup, driveBackup *models.DirectoryData) error {
	slog.Info(fmt.Sprintf("Fetching backup attributes from %s", s.driveProvider), "name", backup.Name)
	attributes, err := s.drive.GetBackupAttributesByID(driveBackup.Identifier)
	if err != nil {
		return err
	}

	slog.Info("Attributes fetched", "name", driveBackup.Name, "size", attributes.Size, "modified", attributes.Modified)

	backup.Drive.Identifier = driveBackup.Identifier
	backup.Drive.Name = driveBackup.Name
	backup.Name = driveBackup.Name
	backup.Date = attributes.Modified
	backup.Size = attributes.Size

	return nil
}

// enforceBackupLimit removes the oldest backups if the number of backups exceed BackupsToKeep
func (s *BackupService) enforceBackupLimit() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.ongoingBackups) > 0 {
		slog.Debug("Backup operation in progress, deferring enforcement of backup limit")
		return nil
	}

	currentCount := len(s.backups)
	if currentCount <= s.config.BackupsToKeep {
		slog.Debug("No backup limit enforcement needed")
		return nil
	}

	slog.Debug("Enforcing backup limit", "CurrentCount", currentCount, "Limit", s.config.BackupsToKeep)

	excessCount := currentCount - s.config.BackupsToKeep
	for i := 0; i < excessCount; i++ {
		oldestBackupIndex := currentCount - 1 - i
		slog.Info("Attempting to delete backup", "name", s.backups[oldestBackupIndex].Name)
		if err := s.DeleteBackup(s.backups[oldestBackupIndex].ID); err != nil {
			slog.Error("Failed to delete backup", "name", s.backups[oldestBackupIndex].Name, "error", err)
			// Consider whether to continue or return on error
		}
	}

	s.backups = s.backups[:s.config.BackupsToKeep]
	slog.Debug("Backup limit enforced", "remaining backups", len(s.backups))
	return s.saveBackupsToFile()
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
				} else {
					if err := s.enforceBackupLimit(); err != nil {
						slog.Error("Error enforcing backup limit", "error", err)
					}
				}
				s.resetTimerForNextBackup()
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

				if err := s.enforceBackupLimit(); err != nil {
					slog.Error("Error enforcing backup limit", "error", err)
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
		if newInterval != s.backupInterval {
			s.backupInterval = newInterval
			s.resetTimerForNextBackup()
			slog.Info("Backup configuration updated", "backupInterval", newInterval.String())
		}

		newBackupsToKeep := s.configService.GetBackupsToKeep()
		if newBackupsToKeep != s.backupsToKeep {
			s.backupsToKeep = newBackupsToKeep
			s.enforceBackupLimit()
			slog.Info("Backup configuration updated", "backupsToKeep", newBackupsToKeep)
		}
	}
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

	now := time.Now().In(s.config.Timezone)
	formattedDate := now.Format("2006-01-02 15:04:05")

	return fmt.Sprintf("Full Backup %s", formattedDate)
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
	id, err := s.drive.UploadFileByPath(fmt.Sprintf("%s.%s", backup.Name, "tar"), path)
	if err != nil {
		backup.UpdateStatus(models.StatusFailed)
		slog.Error(fmt.Sprintf("Error uploading backup to %s", s.driveProvider), err)
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
