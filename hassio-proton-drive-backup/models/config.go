package models

import (
	"log/slog"
	"time"
)

// Config is a struct to represent the configuration of the addon
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

	BackupNameFormat string `json:"backupNameFormat"` // BackupNameFormat is the format to use for the backup name
	BackupInterval   int    `json:"backupInterval"`   // BackupInterval is the interval in days at which backups are performed
	BackupsInHA      int    `json:"backupsInHA"`      // NumberOfBackups is the number of backups to keep
	BackupsOnDrive   int    `json:"backupsOnDrive"`   // NumberOfBackupsOnDrive is the number of backups to keep on the drive
}

// ConfigUpdate is a struct to represent the configuration update
type ConfigUpdate struct {
	BackupNameFormat string `json:"backupNameFormat"`
	BackupInterval   int    `json:"backupInterval"`
	BackupsInHA      int    `json:"backupsInHA"`
	BackupsOnDrive   int    `json:"backupsOnDrive"`
}
