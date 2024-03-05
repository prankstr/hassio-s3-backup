package models

import "time"

// BackupStatus is a custom type to represent the status of the backup
type BackupStatus string

const (
	StatusDeleting  BackupStatus = "DELETING"
	StatusPending   BackupStatus = "PENDING"
	StatusRunning   BackupStatus = "RUNNING"
	StatusCompleted BackupStatus = "COMPLETED"
	StatusSyncing   BackupStatus = "SYNCING"
	StatusFailed    BackupStatus = "FAILED"
	TypeFull        string       = "full"
	TypePartial     string       = "partial"
)

// Backup represents the details and status of a backup process
type Backup struct {
	ID                string       `json:"id"`                // A unique identifier for the backup
	Name              string       `json:"name"`              // Name of the backup
	Status            BackupStatus `json:"status"`            // Current status of the backup
	Date              time.Time    `json:"date"`              // When the backup process started
	Type              string       `json:"type"`              // Type of backup (full or partial)
	ErrorMessage      string       `json:"errorMessage"`      // Error message in case of failure
	Size              float64      `json:"size"`              // Size of the backup file
	Slug              string       `json:"slug"`              // Backup slug from Home assistant
	ProtonLink        string       `json:"protonLink"`        // Proton Drive Link to file
	OnlyInHA          bool         `json:"onlyInHa"`          // If backup should only be in HA
	OnlyOnProtonDrive bool         `json:"onlyOnProtonDrive"` // If backup should only be on proton drive
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
