package models

import "time"

type HassBackup struct {
	Slug       string        `json:"slug"`
	Date       time.Time     `json:"date"`
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Size       float64       `json:"size"`
	Protected  bool          `json:"protected"`
	Location   string        `json:"location"`
	Compressed bool          `json:"compressed"`
	Content    BackupContent `json:"content"`
}

type BackupContent struct {
	HomeAssistant bool     `json:"homeassistant"`
	Addons        []string `json:"addons"`
	Folders       []string `json:"folders"`
}

type HassBackupResponse struct {
	Result string `json:"result"`
	Data   struct {
		Backups []*HassBackup `json:"backups"`
	} `json:"data"`
}
type HassioResponseData struct {
	Slug         string `json:"slug"`
	IngressEntry string `json:"ingress_entry"`
}

type HassioResponse struct {
	Result  string `json:"result"`
	Message string `json:"message"`
	Data    HassioResponseData
}
