package models

import (
	"log/slog"
	"time"
)

// Config is a struct to represent the configuration of the addon
type Config struct {
	Timezone         *time.Location
	S3               S3Config
	Storj            StorjConfig
	DataDirectory    string
	StorageBackend   string `json:"storageBackend"`
	BackupDirectory  string
	SupervisorToken  string
	BackupNameFormat string `json:"backupNameFormat"`
	IngressPath      string
	Hostname         string
	LogLevel         slog.Level
	BackupInterval   int `json:"backupInterval"`
	BackupsInHA      int `json:"backupsInHA"`
	BackupsInStorage int `json:"backupsInStorage"`
	Debug            bool
}

type StorjConfig struct {
	AccessGrant string
	BucketName  string
}

type S3Config struct {
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Endpoint        string
}

type Credentials struct {
	Username        string
	Password        string
	AccessGrant     string
	AccessKeyID     string
	SecretAccessKey string
}
