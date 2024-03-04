package models

import (
	"log/slog"
	"time"
)

type Config struct {
	Hostname        string         // Hostname is the hostname of the machine running the addon
	IngressPath     string         // IngressPath is the path where the ingress endpoint is exposed
	SupervisorToken string         // SupervisorToken is the token used to authenticate with the Supervisor API
	Debug           bool           // Debug is a flag that enables debug mode
	BackupDirectory string         // BackupDirectory is the directory on the drive where backups are stored
	DataDirectory   string         // DataDirectory is the directory where the config and backup tracking files are stored
	LogLevel        slog.Level     // LogLevel is the log level to use
	Timezone        *time.Location // Timezone that will be used to for backup times

	ProtonDriveUser     string
	ProtonDrivePassword string

	BackupInterval int `json:"backupInterval"` // BackupInterval is the interval in days at which backups are performed
	BackupsToKeep  int `json:"backupsToKeep"`  // NumberOfBackups is the number of backups to keep
}

type ConfigUpdate struct {
	BackupInterval int `json:"backupInterval"`
	BackupsToKeep  int `json:"backupsToKeep"`
}
