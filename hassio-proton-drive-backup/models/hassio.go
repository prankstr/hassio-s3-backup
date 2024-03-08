package models

import "time"

// HassBackup represents the details of a backup in Home Assistant
type HassBackup struct {
	Slug       string        `json:"slug"`       // A unique identifier for the backup
	Date       time.Time     `json:"date"`       // When the backup process started
	Name       string        `json:"name"`       // Name of the backup
	Type       string        `json:"type"`       // Type of the backup, full or partial
	Size       float64       `json:"size"`       // Size of the backup in MB
	Protected  bool          `json:"protected"`  // If the backup is protected
	Location   string        `json:"location"`   // Location of the backup
	Compressed bool          `json:"compressed"` // If the backup is compressed
	Content    BackupContent `json:"content"`    // Content of the backup
}

// BackupContent represents the content of a backup
type BackupContent struct {
	HomeAssistant bool     `json:"homeassistant"`
	Addons        []string `json:"addons"`
	Folders       []string `json:"folders"`
}

// HassBackupResponse represents the response from Home Assistant
type HassBackupResponse struct {
	Result string `json:"result"`
	Data   struct {
		Backups []*HassBackup `json:"backups"`
	} `json:"data"`
}

// HassioResponseData represents the data in the response from Home Assistant
type HassioResponseData struct {
	Slug         string `json:"slug"`
	IngressEntry string `json:"ingress_entry"`
}

// HassioResponse represents the response from Home Assistant
type HassioResponse struct {
	Result  string `json:"result"`
	Message string `json:"message"`
	Data    HassioResponseData
}
