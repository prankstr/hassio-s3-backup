package backup

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hassio-proton-drive-backup/internal/config"
	"hassio-proton-drive-backup/internal/hassio"
	"hassio-proton-drive-backup/internal/s3"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
)

// status is a custom type to represent the status of the backup
type status string

const (
	StatusDeleting    status = "DELETING"    // Backup is being deleted
	StatusPending     status = "PENDING"     // Backup is initialized but no action taken
	StatusRunning     status = "RUNNING"     // Backup is being created in Home Assistant
	StatusSynced      status = "SYNCED"      // Backup is present in both Home Assistant and S3
	StatusHAOnly      status = "HAONLY"      // Backup is only present in Home Assistant
	StatusS3Only      status = "S3ONLY"      // Backup is only present in S3
	StatusSyncing     status = "SYNCING"     // Backup is being uploaded to S3
	StatusDownloading status = "DOWNLOADING" // Backup is being downloaded from S3
	StatusFailed      status = "FAILED"      // Backup process failed somewhere
)

// Backup represents the details and status of a backup process
type Backup struct {
	Date         time.Time      `json:"date"`
	S3           *s3.Object     `json:"s3"`
	HA           *hassio.Backup `json:"ha"`
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Status       status         `json:"status"`
	ErrorMessage string         `json:"errorMessage"`
	Pinned       bool           `json:"pinned"`
}

// UpdateStatus updates the status of the backup
func (b *Backup) UpdateStatus(status status) {
	b.Status = status
}

// Service handles backup operations and synchronization
type Service struct {
	s3Client      *minio.Client
	hassioClient  *hassio.Client
	configService *config.Service
	config        *config.Options
	backups       []*Backup
	mutex         sync.Mutex
}

var (
	backupTimer            *time.Timer
	syncTicker             *time.Ticker
	syncInterval           time.Duration
	stopBackupChan         chan struct{}
	stopSyncChan           chan struct{}
	nextBackupCalculatedAt time.Time
	nextBackupIn           time.Duration
	ongoingBackups         map[string]struct{}
)

func init() {
	stopBackupChan = make(chan struct{})
	stopSyncChan = make(chan struct{})

	backupTimer = time.NewTimer(time.Hour)
	backupTimer.Stop()

	syncInterval = 1 * time.Hour

	ongoingBackups = make(map[string]struct{})
}

// NewService creates a new Service instance
func NewService(s3Client *minio.Client, configService *config.Service) *Service {
	hassioClient := hassio.NewService(configService.Config.SupervisorToken)

	service := &Service{
		hassioClient:  hassioClient,
		s3Client:      s3Client,
		configService: configService,
		config:        configService.Config,
	}

	// Initial load and sync of backups
	service.loadBackupsFromFile()
	service.syncBackups()

	// Start scheduled backups and syncs
	go service.startBackupScheduler()
	go service.startBackupSyncScheduler()
	go service.listenForConfigChanges(configService.ConfigChangeChan)

	return service
}

// PerformBackup creates a new backup and uploads it to S3
func (s *Service) PerformBackup(name string) error {
	if len(ongoingBackups) > 0 {
		err := errors.New("another backup is already in progress")
		slog.Error(err.Error())
		return err
	}

	backup := s.initializeBackup(name)

	// Track ongoing backups to avoid syncing or any other manipulation in the meantime
	ongoingBackups[backup.ID] = struct{}{}
	defer delete(ongoingBackups, backup.ID)

	backup.UpdateStatus(StatusRunning)
	slug, err := s.hassioClient.BackupFull(backup.Name)
	if err != nil {
		backup.ErrorMessage = err.Error()
		backup.UpdateStatus(StatusFailed)

		err = fmt.Errorf("backup creation in home assistant failed: %v", err)
		return err
	}

	slog.Debug("backup created in home assistant", "name", backup.Name, "slug", backup.HA.Slug)
	backup.HA.Slug = slug

	err = s.syncBackupToS3(backup)
	if err != nil {
		return err
	}
	slog.Debug("backup uploaded to s3", "name", backup.Name)

	backup.UpdateStatus(StatusSynced)
	delete(ongoingBackups, backup.ID)
	slog.Info("backup successfully created and synced", "name", backup.Name)

	if err := s.syncBackups(); err != nil {
		slog.Error("error syncing backups", "error", err)
	}

	return nil
}

// DeleteBackup deletes a backup from all sources
func (s *Service) DeleteBackup(id string) error {
	index, backup := s.getBackupByID(id)

	// Delete backup from Home Assistant
	backup.UpdateStatus(StatusDeleting)

	if backup.HA != nil && *backup.HA != (hassio.Backup{}) {
		slog.Debug("deleting backup from home assistant", "name", backup.Name)
		err := s.hassioClient.DeleteBackup(backup.HA.Slug)
		if err != nil {
			slog.Error("failed to delete backup in home assistant", "name", backup.Name, "error", err)
		}
	}

	// Delete backup from S3

	if backup.S3 != nil && *backup.S3 != (s3.Object{}) {
		slog.Debug("deleting backup from s3", "backup", backup)
		err := s.s3Client.RemoveObject(context.Background(), s.config.S3.Bucket, backup.S3.Key, minio.RemoveObjectOptions{})
		if err != nil {
			return handleBackupError(s, "failed to delete backup from s3", backup, err)
		}
	}

	// Remove backup from local list
	s.backups = append(s.backups[:index], s.backups[index+1:]...)

	// Save the updated backup state to file
	if err := s.saveBackupsToFile(); err != nil {
		slog.Error("error saving backup state after deletion", "error", err)
		return err
	}

	// Reset the timer for the next backup
	s.resetTimerForNextBackup()

	slog.Info("backup deleted", "name", backup.Name)
	return nil
}

// RestoreBackup calls Home Assistant to restore a backup
// Note: might not be needed, as the restore can be done from the Home Assistant UI
func (s *Service) RestoreBackup(id string) error {
	_, backup := s.getBackupByID(id)
	err := s.hassioClient.RestoreBackup(backup.HA.Slug)
	if err != nil {
		return fmt.Errorf("failed to restore backup in home assistant: %v", err)
	}

	slog.Info("restored to backup", "name", backup.Name)
	return nil
}

// DownloadBackup downloads a backup from S3 to Home Assistant
func (s *Service) DownloadBackup(id string) error {
	_, backup := s.getBackupByID(id)

	slog.Debug("downloading backup to home assistant", "name", backup.Name)
	backup.UpdateStatus(StatusDownloading)

	object, err := s.s3Client.GetObject(context.Background(), s.config.S3.Bucket, backup.S3.Key, minio.GetObjectOptions{})
	if err != nil {
		slog.Error("failed to get backup from s3", "name", backup.Name, "error", err)
		backup.UpdateStatus(StatusS3Only)
		return err
	}
	defer object.Close()

	err = s.hassioClient.UploadBackup(object)
	if err != nil {
		slog.Error("failed to upload backup to home assistant", "name", backup.Name, "error", err)
		backup.UpdateStatus(StatusS3Only)
		return err
	}

	slog.Info("backup downloaded", "name", backup.Name)
	s.syncBackups()

	return nil
}

// PinBackup pins a backup to prevent it from being deleted
func (s *Service) PinBackup(id string) error {
	_, backup := s.getBackupByID(id)
	backup.Pinned = true

	slog.Info("backup pinned", "name", backup.Name)

	return s.saveBackupsToFile()
}

// UnpinBackup unpins a backup to allow it to be deleted
func (s *Service) UnpinBackup(id string) error {
	_, backup := s.getBackupByID(id)
	backup.Pinned = false

	slog.Info("backup unpinned", "name", backup.Name)

	return s.saveBackupsToFile()
}

// ListBackups returns the list of backups in memory
func (s *Service) ListBackups() []*Backup {
	return s.backups
}

// TimeUntilNextBackup returns the time until the next backup in milliseconds
func (s *Service) TimeUntilNextBackup() int64 {
	return time.Until(nextBackupCalculatedAt.Add(nextBackupIn)).Milliseconds()
}

// NameExists checks if a backup with the given name exists
func (s *Service) NameExists(name string) bool {
	generatedName := generateBackupName(name, s.config.BackupNameFormat, s.config.Timezone)

	for _, backup := range s.backups {
		if backup.Name == generatedName {
			return true
		}
	}

	return false
}

// ResetBackups resets the local state of backups
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

// syncBackups synchronizes the backups by performing the following steps
func (s *Service) syncBackups() error {
	// Cancel if there is an ongoing backup
	if len(ongoingBackups) > 0 {
		slog.Debug("skipping synchronization due to ongoing backup operations.")
		return nil
	}

	// Reset timer when this function returns
	defer s.resetTimerForNextBackup()

	// Take an initial snapshot of the state
	initialState, err := s.calculateBackupsHash()
	if err != nil {
		return err
	}

	// Create a map of backups for easy access
	backupMap := make(map[string]*Backup)
	for _, backup := range s.backups {
		backupMap[backup.Name] = backup
		// Nil out HA and S3
		// This will delete the backup from the map if it's not found in HA or S3 during the sync
		backup.HA = nil
		backup.S3 = nil
	}

	// Keep HA backups up to date
	err = s.updateHABackups(backupMap)
	if err != nil {
		return err
	}

	// Keep S3 backups up to date
	err = s.updateS3Backups(backupMap)
	if err != nil {
		return err
	}

	// Mark backups for deletion if needed
	err = s.deleteExcessBackups()
	if err != nil {
		return err
	}

	// Update statuses and sync backups to S3 if needed
	for _, backup := range s.backups {
		backupInHA, backupInS3 := backup.HA != nil, backup.S3 != nil
		if backupInHA && backupInS3 {
			backup.UpdateStatus(StatusSynced)
		} else if backupInHA {
			backup.UpdateStatus(StatusHAOnly)
		} else if backupInS3 {
			backup.UpdateStatus(StatusS3Only)
		}
	}

	if err := s.ensureS3Backups(); err != nil {
		return err
	}

	// Take a final snapshot of the state
	finalState, err := s.calculateBackupsHash()
	if err != nil {
		return err
	}

	// Compare initial and final state to determine if anything was done
	if initialState == finalState {
		slog.Info("nothing to do")
	}

	// Sort and save backups
	sort.Slice(s.backups, func(i, j int) bool {
		return s.backups[i].Date.After(s.backups[j].Date)
	})

	if err := s.saveBackupsToFile(); err != nil {
		slog.Error("error saving backup state after backup operation", "error", err)
		return err
	}

	return nil
}

// ensureS3Backups syncs the required number of backups to S3
func (s *Service) ensureS3Backups() error {
	haOnlyBackups := []*Backup{}
	s3Backups := 0

	for _, backup := range s.backups {
		if !backup.Pinned {
			switch backup.Status {
			case StatusSynced, StatusS3Only:
				s3Backups++
			case StatusHAOnly:
				haOnlyBackups = append(haOnlyBackups, backup)
			}
		}
	}

	var uploadCount int
	if s.config.BackupsInS3 > 0 {
		uploadCount = s.config.BackupsInS3 - s3Backups
	} else {
		uploadCount = len(haOnlyBackups)
	}

	if uploadCount > 0 && uploadCount > len(haOnlyBackups) {
		uploadCount = len(haOnlyBackups)
	}

	for i := 0; i < uploadCount; i++ {
		backup := haOnlyBackups[i]
		if err := s.syncBackupToS3(backup); err != nil {
			return err
		}
	}

	return nil
}

// updateHABackups adds Home Assistant backups to the backup map if they don't exist by name
func (s *Service) updateHABackups(backupMap map[string]*Backup) error {
	haBackups, err := s.hassioClient.ListBackups()
	if err != nil {
		return err
	}

	if len(haBackups) == 0 {
		slog.Debug("no backups found in home assistant")
		return nil
	}

	for _, haBackup := range haBackups {
		if haBackup.Type == "partial" {
			continue // Skip partial backups
		}

		if _, exists := backupMap[haBackup.Name]; !exists {
			slog.Info("found untracked backup in home assistant", "name", haBackup.Name)

			backup := s.initializeBackup(haBackup.Name)
			backup.HA = haBackup
			backup.Date = haBackup.Date.In(s.config.Timezone)

			backupMap[haBackup.Name] = backup
		} else {
			backupMap[haBackup.Name].HA = haBackup
		}
	}

	return nil
}

// updateS3Backups adds backups found in S3 to the backup map if they don't exist by name
func (s *Service) updateS3Backups(backupMap map[string]*Backup) error {
	s3Backups := []*s3.Object{}
	objectCh := s.s3Client.ListObjects(context.Background(), s.config.S3.Bucket, minio.ListObjectsOptions{})
	for object := range objectCh {
		if object.Err != nil {
			slog.Error("could not list objects in s3: %v", "error", object.Err)
			return fmt.Errorf("could not list objects: %v", object.Err)
		}

		s3Backups = append(s3Backups, &s3.Object{
			Key:      object.Key,
			Size:     float64(object.Size) / (1024 * 1024), // convert bytes to MB
			Modified: object.LastModified,
		})
	}

	if len(s3Backups) == 0 {
		slog.Debug("no backups found in s3")
		return nil
	}

	for _, s3Backup := range s3Backups {
		name := strings.TrimSuffix(s3Backup.Key, ".tar")

		if _, exists := backupMap[name]; !exists {
			slog.Info("found untracked backup in s3", "name", s3Backup.Key)
			backup := s.initializeBackup(name)

			backup.S3 = s3Backup
			backup.Date = s3Backup.Modified

		} else {
			backupMap[name].S3 = s3Backup
		}
	}

	return nil
}

// deleteExcessBackups marks the oldest excess backups for deletion based on the given limit and returns true if any were marked for deletion
func (s *Service) deleteExcessBackups() error {
	// Get backups that aren't pinned or failed
	backups := []*Backup{}

	for _, backup := range s.backups {
		if !backup.Pinned && backup.Status != StatusFailed {
			backups = append(backups, backup)
		}
	}

	// Sort non-pinned backups by date, oldest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Date.Before(backups[j].Date)
	})

	// Mark excess backups in HA for deletion
	if s.config.BackupsInHA > 0 {
		haBackups := []*Backup{}
		var emptyHA hassio.Backup

		for _, backup := range backups {
			if backup.HA != nil && *backup.HA != emptyHA {
				haBackups = append(haBackups, backup)
			}
		}

		// Retain the most recent HA backups
		if len(haBackups) > s.config.BackupsInHA {
			for i := 0; i < len(haBackups)-s.config.BackupsInHA; i++ {
				if !haBackups[i].Pinned {
					if err := s.hassioClient.DeleteBackup(haBackups[i].HA.Slug); err != nil {
						return err
					}

					haBackups[i].HA = nil

					slog.Info("deleted backup from home assistant", "name", haBackups[i].Name)
				}
			}
		} else {
			slog.Debug("skipping deletion for Home Assistant backups; limit is set to 0.")
		}
	}

	// Mark excess backups in S3 for deletion
	if s.config.BackupsInS3 > 0 {

		s3Backups := []*Backup{}
		for _, backup := range backups {
			if backup.S3 != nil {
				s3Backups = append(s3Backups, backup)
			}
		}

		if len(s3Backups) > s.config.BackupsInS3 {
			// Mark the oldest S3 backups for deletion
			for i := 0; i < len(s3Backups)-s.config.BackupsInS3; i++ {
				if !s3Backups[i].Pinned {
					if err := s.s3Client.RemoveObject(context.Background(), s.config.S3.Bucket, s3Backups[i].S3.Key, minio.RemoveObjectOptions{}); err != nil {
						return err
					}

					s3Backups[i].S3 = nil

					slog.Info("deleted backup from S3", "name", s3Backups[i].Name)
				}
			}
		} else {
			slog.Debug("skipping deletion for S3 backups; limit is set to 0.")
		}
	}

	// Delete backups from the local map after ensuring HA and S3 are up to date
	backupsToKeep := []*Backup{}
	for _, backup := range s.backups {
		if backup.HA != nil || backup.S3 != nil || backup.Status == StatusFailed {
			backupsToKeep = append(backupsToKeep, backup)
		}
	}

	s.backups = backupsToKeep

	return nil
}

// initializeBackup returns a new internal backup object
func (s *Service) initializeBackup(name string) *Backup {
	generatedName := generateBackupName(name, s.config.BackupNameFormat, s.config.Timezone)

	backup := &Backup{
		ID:     base64.RawURLEncoding.EncodeToString([]byte(generatedName)),
		Name:   generatedName,
		Date:   time.Now().In(s.config.Timezone),
		Status: StatusPending,
		S3:     new(s3.Object),
		HA:     new(hassio.Backup),
	}

	s.backups = append([]*Backup{backup}, s.backups...)

	slog.Debug("new backup initialized", "name", backup.Name, "status", backup.Status)
	return backup
}

// generateBackupName generates a backup name based on the provided format and timezone
func generateBackupName(requestName string, format string, timezone *time.Location) string {
	if requestName != "" {
		return requestName
	}

	now := time.Now().In(timezone)
	format = strings.ReplaceAll(format, "{year}", now.Format("2006"))
	format = strings.ReplaceAll(format, "{month}", now.Format("01"))
	format = strings.ReplaceAll(format, "{day}", now.Format("02"))
	format = strings.ReplaceAll(format, "{hr24}", now.Format("15"))
	format = strings.ReplaceAll(format, "{min}", now.Format("04"))
	format = strings.ReplaceAll(format, "{sec}", now.Format("05"))

	return format
}

// syncBackupToS3 uploads a backup to the remote drive if needed
func (s *Service) syncBackupToS3(backup *Backup) error {
	if backup.S3 != nil {
		_, err := s.s3Client.StatObject(context.Background(), s.config.S3.Bucket, backup.Name, minio.StatObjectOptions{})
		if err == nil {
			return nil
		}
	}

	slog.Debug("syncing backup to s3", "name", backup.Name)
	backup.UpdateStatus(StatusSyncing)
	_, err := s.uploadBackupToS3(backup)
	if err != nil {
		backup.UpdateStatus(StatusFailed)
		backup.ErrorMessage = err.Error()

		if err := s.saveBackupsToFile(); err != nil {
			slog.Error("error saving backup state after backup operation", "error", err)
			return err
		}

		return handleBackupError(s, "failed to sync backup to s3", backup, err)
	}

	s.updateS3BackupDetails(backup)

	backup.UpdateStatus(StatusSynced)

	return nil
}

// uploadBackupToS3 uploads a backup from Home Assistant to the remote drive
func (s *Service) uploadBackupToS3(backup *Backup) (string, error) {
	ctx := context.Background()
	contentType := "application/octet-stream"

	objectName := fmt.Sprintf("%s.%s", backup.Name, "tar")
	path := fmt.Sprintf("%s/%s.%s", "/backup", backup.HA.Slug, "tar")

	slog.Debug("uploading backup to s3", "name", backup.Name)
	info, err := s.s3Client.FPutObject(ctx, s.config.S3.Bucket, objectName, path, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}

	return info.Key, nil
}

// handleBackupError takes a standard set of actions when a backup error occurs
func handleBackupError(s *Service, errMsg string, backup *Backup, err error) error {
	slog.Error(errMsg, "error", err)
	if backup != nil {
		backup.UpdateStatus(StatusFailed)
		backup.ErrorMessage = err.Error()
		s.saveBackupsToFile() // Best effort to save state
	}
	return fmt.Errorf("%s: %v", errMsg, err)
}

// startBackupScheduler starts a goroutine that will perform backups on a timer
func (s *Service) startBackupScheduler() {
	s.resetTimerForNextBackup()

	go func() {
		for {
			select {
			case <-backupTimer.C:
				slog.Info("performing scheduled backup")

				if err := s.PerformBackup(""); err != nil {
					slog.Error("failed to perform scheduled backup", "error", err)
				}
			case <-stopBackupChan:
				slog.Info("stopping backup scheduler")
				return
			}
		}
	}()
}

// startBackupSyncScheduler starts a goroutine that will perform syncs on a timer
func (s *Service) startBackupSyncScheduler() {
	syncTicker = time.NewTicker(syncInterval)

	go func() {
		for {
			select {
			case <-syncTicker.C:
				slog.Info("performing scheduled backup sync")

				if err := s.syncBackups(); err != nil {
					slog.Error("error performing backup sync", "error", err)
				}
			case <-stopSyncChan:
				slog.Info("stopping sync scheduler")
				syncTicker.Stop()
				return
			}
		}
	}()
}

// calculateDurationUntilNextBackup calculates the duration until the next backup should occur
func (s *Service) calculateDurationUntilNextBackup() time.Duration {
	latestBackup := s.getLatestBackup()
	if latestBackup == nil {
		return 1 * time.Second
	}

	elapsed := time.Since(latestBackup.Date)
	if elapsed >= time.Duration(s.config.BackupInterval)*24*time.Hour {
		return 1 * time.Second
	}

	return time.Duration(s.config.BackupInterval)*24*time.Hour - elapsed
}

// resetTimerForNextBackup sets the timer for the next backup
func (s *Service) resetTimerForNextBackup() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	nextBackupIn = s.calculateDurationUntilNextBackup()
	nextBackupCalculatedAt = time.Now()

	if !backupTimer.Stop() && backupTimer != nil {
		select {
		case <-backupTimer.C: // Drain the channel
		default:
		}
	}

	backupTimer.Reset(nextBackupIn)
	slog.Debug(fmt.Sprintf("next backup in %s", nextBackupIn.String()))
}

// listenForConfigChanges listens for changes to certain config values and takes action when the config changes
func (s *Service) listenForConfigChanges(configChan <-chan *config.Options) {
	for range configChan {
		s.syncBackups()
	}
}

// loadBackupsFromFile populates the initial list of backups from a file on disk
func (s *Service) loadBackupsFromFile() {
	path := "/data/backups.json"
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("error loading backups from file", "error", err)
		return
	}

	if err := json.Unmarshal(data, &s.backups); err != nil {
		slog.Error("error unmarshaling backups", "error", err)
	}
}

// saveBackupsToFile persists the list of backups to a file on disk
func (s *Service) saveBackupsToFile() error {
	data, err := json.Marshal(s.backups)
	if err != nil {
		return err
	}

	err = os.WriteFile("/data/backups.json", data, 0644)
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

// getBackupByID finds and returns the backup by ID
func (s *Service) getBackupByID(id string) (int, *Backup) {
	for i, b := range s.backups {
		if b.ID == id {
			return i, b
		}
	}

	return -1, nil
}

// updateS3BackupDetails updates the backup with information from S3
func (s *Service) updateS3BackupDetails(backup *Backup) error {
	slog.Debug("fetching backup attributes from s3", "name", backup.Name)

	objectName := backup.Name + ".tar"
	object, err := s.s3Client.StatObject(context.Background(), s.config.S3.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return fmt.Errorf("could not open object: %v", err)
	}

	attributes := &s3.Object{
		Key:      objectName,
		Size:     float64(object.Size) / (1024 * 1024), // convert bytes to MB
		Modified: object.LastModified,
	}

	slog.Debug("attributes fetched", "key", attributes.Key, "size", attributes.Size, "modified", attributes.Modified)

	backup.S3 = attributes

	return nil
}

// calculateBackupsHash returns a hash of the backup array
func (s *Service) calculateBackupsHash() (string, error) {
	h := sha256.New()
	for _, backup := range s.backups {
		backupJSON, err := json.Marshal(backup)
		if err != nil {
			return "", err
		}
		h.Write(backupJSON)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
