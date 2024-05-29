package models

import "time"

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
