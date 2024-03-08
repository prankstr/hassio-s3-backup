package models

import "time"

// BackupStatus is a custom type to represent the status of the backup
type BackupStatus string

const (
	StatusDeleting  BackupStatus = "DELETING"  // Backup in being deleted
	StatusPending   BackupStatus = "PENDING"   // Backup is initalized but no action taken
	StatusRunning   BackupStatus = "RUNNING"   // Backup is being created in Home Assistant
	StatusSynced    BackupStatus = "SYNCED"    // Backup is present in both Home Assistant and drive
	StatusHAOnly    BackupStatus = "HAONLY"    // Backup is only present in  Home Assistant
	StatusDriveOnly BackupStatus = "DRIVEONLY" // Backup is only present on drive
	StatusSyncing   BackupStatus = "SYNCING"   // Backups is being uploaded to drive
	StatusFailed    BackupStatus = "FAILED"    // Backup process failed somewhere
)

// Backup represents the details and status of a backup process
type Backup struct {
	ID                string         `json:"id"`                // A unique identifier for the backup
	Name              string         `json:"name"`              // Name of the backup
	Status            BackupStatus   `json:"status"`            // Current status of the backup
	Date              time.Time      `json:"date"`              // When the backup process started
	Size              float64        `json:"size"`              // Size of the backup in MB
	ErrorMessage      string         `json:"errorMessage"`      // Error message in case of failure
	MarkedForDeletion bool           `json:"markedForDeletion"` // Marked for deletion
	Drive             *DirectoryData `json:"proton"`            // Drive backup details
	HA                *HassBackup    `json:"ha"`                // Home Assistant backup details
}

// HAData is a selection of metadata from the HA backups
type HAData struct {
	Slug string    `json:"slug"`
	Date time.Time `json:"date"`
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
