package models

import "time"

// BackupStatus is a custom type to represent the status of the backup
type BackupStatus string

const (
	StatusDeleting  BackupStatus = "DELETING"
	StatusPending   BackupStatus = "PENDING"
	StatusRunning   BackupStatus = "RUNNING"
	StatusSynced    BackupStatus = "SYNCED"
	StatusHAOnly    BackupStatus = "HAONLY"
	StatusDriveOnly BackupStatus = "DRIVEONLY"
	StatusSyncing   BackupStatus = "SYNCING"
	StatusFailed    BackupStatus = "FAILED"
	TypeFull        string       = "full"
	TypePartial     string       = "partial"
)

type HAData struct {
	Slug string    `json:"slug"`
	Date time.Time `json:"date"`
	Size float64   `json:"size"`
}

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
