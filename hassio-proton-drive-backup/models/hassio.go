package models

import "time"

// HassBackup represents the details of a backup in Home Assistant
type HassBackup struct {
	Date       time.Time     `json:"date"`
	Slug       string        `json:"slug"`
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Location   string        `json:"location"`
	Content    BackupContent `json:"content"`
	Size       float64       `json:"size"`
	Protected  bool          `json:"protected"`
	Compressed bool          `json:"compressed"`
}

// BackupContent represents the content of a backup
type BackupContent struct {
	Addons        []string `json:"addons"`
	Folders       []string `json:"folders"`
	HomeAssistant bool     `json:"homeassistant"`
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
