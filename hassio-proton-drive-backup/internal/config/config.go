package config

import (
	"encoding/json"
	"fmt"
	"hassio-proton-drive-backup/internal/hassio"
	"hassio-proton-drive-backup/internal/storage"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
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

// ConfigService is a struct to handle the application configuration
type ConfigService struct {
	Config           *Config
	Creds            *storage.Credentials
	ConfigChangeChan chan *Config
}

// logLevels is a map of string to slog.Level
var logLevels map[string]slog.Level = map[string]slog.Level{
	"Error": slog.LevelError,
	"Warn":  slog.LevelWarn,
	"Info":  slog.LevelInfo,
	"Debug": slog.LevelDebug,
}

// New returns a new Config struct
func NewConfigService() *ConfigService {
	// Read configuration from file
	config, err := readConfigFromFile("/data/config.json")
	if err != nil {
		// Handle error. This could be logging the error and continuing with defaults
		slog.Error("Error reading config file: ", err)
		config = &Config{} // Initialize with an empty config
	}
	creds := &storage.Credentials{} // Initialize with an empty config

	// Set defaults or override with environment variables if they are set
	config.Hostname = getEnvOrDefault("HOSTNAME", config.Hostname, "localhost:9101")
	config.SupervisorToken = getEnvOrDefault("SUPERVISOR_TOKEN", config.SupervisorToken, "")
	config.StorageBackend = getEnvOrDefault("STORAGE_BACKEND", config.StorageBackend, "")
	config.BackupDirectory = getEnvOrDefault("BACKUP_DIRECTORY", config.BackupDirectory, "Home Assistant Backups")
	config.DataDirectory = getEnvOrDefault("DATA_DIRECTORY", config.DataDirectory, "/data")
	config.BackupsInHA = getEnvOrDefaultInt("BACKUPS_IN_HA", config.BackupsInHA, 0)
	config.BackupNameFormat = getEnvOrDefault("BACKUP_NAME_FORMAT", config.BackupNameFormat, "Full Backup {year}-{month}-{day} {hr24}:{min}:{sec}")
	config.BackupsInStorage = getEnvOrDefaultInt("BACKUPS_IN_STORAGE", config.BackupsInStorage, 0)
	config.BackupInterval = getEnvOrDefaultInt("BACKUP_INTERVAL", config.BackupInterval, 3)

	// S3 Config
	if config.StorageBackend == storage.S3 {
		config.S3.AccessKeyID = getEnvOrDefault("S3_ACCESS_KEY_ID", config.S3.AccessKeyID, "")
		config.S3.SecretAccessKey = getEnvOrDefault("S3_SECRET_ACCESS_KEY", config.S3.SecretAccessKey, "")
		config.S3.BucketName = getEnvOrDefault("S3_BUCKET_NAME", config.S3.BucketName, "")
		config.S3.Endpoint = getEnvOrDefault("S3_ENDPOINT", config.S3.Endpoint, "")
	}

	// Storj Config
	if config.StorageBackend == storage.STORJ {
		config.Storj.AccessGrant = getEnvOrDefault("STORJ_ACCESS_GRANT", config.Storj.AccessGrant, "")
		config.Storj.BucketName = getEnvOrDefault("STORJ_BUCKET_NAME", config.Storj.BucketName, "")
	}
	//
	creds.Username = getEnvOrDefault("PROTON_DRIVE_USER", "", "")
	creds.Password = getEnvOrDefault("PROTON_DRIVE_PASSWORD", "", "")

	defaultTimezone := "UTC"
	timezoneStr := getEnvOrDefault("TZ", config.Timezone.String(), defaultTimezone)
	config.Timezone, err = time.LoadLocation(timezoneStr)
	if err != nil {
		slog.Error("invalid time zone, defaulting to UTC: %v", err)
		config.Timezone, _ = time.LoadLocation(defaultTimezone)
	}

	logLevelStr := getEnvOrDefault("LOG_LEVEL", stringFromSlogLevel(config.LogLevel), "Debug")
	if level, exists := logLevels[logLevelStr]; exists {
		config.LogLevel = level
	} else {
		slog.Error("Invalid log level specified. Using default level.")
		config.LogLevel = logLevels["Info"]
	}

	// Handle the debug setting
	debugStr := getEnvOrDefault("DEBUG", strconv.FormatBool(config.Debug), "true")
	debug, err := strconv.ParseBool(debugStr)
	if err == nil {
		config.Debug = debug
	} else {
		slog.Error("Cannot parse the DEBUG variable")
	}

	// Handle ingress entry
	ingressEntry := getIngressEntry(config.SupervisorToken)
	config.IngressPath = ingressEntry

	writeConfigToFile(config)

	return &ConfigService{
		Config:           config,
		Creds:            creds,
		ConfigChangeChan: make(chan *Config),
	}
}

// NotifyConfigChange sends a new config to the configChangeChan
func (cs *ConfigService) NotifyConfigChange(newConfig *Config) {
	slog.Info("Config updated, notifying")
	cs.ConfigChangeChan <- newConfig
}

// GetConfig returns the current application configuration.
func (cs *ConfigService) GetConfig() *Config {
	return cs.Config
}

// GetBackupDirectory returns the directory where backups are stored
func (cs *ConfigService) GetBackupInterval() time.Duration {
	return (time.Duration(cs.Config.BackupInterval) * time.Hour) * 24
}

func (cs *ConfigService) GetS3BucketName() string {
	return cs.Config.S3.BucketName
}

func (cs *ConfigService) GetS3Endpoint() string {
	return cs.Config.S3.Endpoint
}

func (cs *ConfigService) GetS3AccessKeyID() string {
	return cs.Config.S3.AccessKeyID
}

func (cs *ConfigService) GetS3SecretAccessKey() string {
	return cs.Config.S3.SecretAccessKey
}

func (cs *ConfigService) GetStorjBucketName() string {
	return cs.Config.Storj.BucketName
}

func (cs *ConfigService) GetCredentials() *storage.Credentials {
	return cs.Creds
}

// GetBackupNameFormat returns the format to use for the backup name
func (cs *ConfigService) GetBackupNameFormat() string {
	return cs.Config.BackupNameFormat
}

// GetBackupsToKeep returns the number of backups to keep
func (cs *ConfigService) GetBackupsInHA() int {
	return cs.Config.BackupsInHA
}

// GetBackupsOnDrive returns the number of backups to keep on the drive
func (cs *ConfigService) GetBackupsInStorage() int {
	return cs.Config.BackupsInStorage
}

// GetProtonDriveUser returns the ProtonDrive username
func (cs *ConfigService) GetProtonDriveUser() string {
	return cs.Creds.Username
}

// GetProtonDrivePassword returns the ProtonDrive password
func (cs *ConfigService) GetProtonDrivePassword() string {
	return cs.Creds.Password
}

// SetBackupsToKeep sets the number of backups to keep
func (cs *ConfigService) SetBackupInterval(interval int) {
	cs.Config.BackupInterval = interval

	writeConfigToFile(cs.Config)
}

// SetBackupsToKeep sets the number of backups to keep
func (cs *ConfigService) UpdateConfigFromAPI(configRequest Config) error {
	cs.Config.BackupNameFormat = configRequest.BackupNameFormat
	cs.Config.BackupInterval = configRequest.BackupInterval
	cs.Config.BackupsInHA = configRequest.BackupsInHA
	cs.Config.BackupsInStorage = configRequest.BackupsInStorage

	cs.NotifyConfigChange(cs.Config)

	writeConfigToFile(cs.Config)
	return nil
}

// Helper function to get environment variable or return a default
func getEnvOrDefault(key string, currentValue, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	if currentValue != "" {
		return currentValue
	}

	return defaultValue
}

// Helper function to get environment variable or return a default
func getEnvOrDefaultInt(key string, currentValue, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}

	if currentValue != 0 {
		return currentValue
	}

	return defaultValue
}

// Helper function to convert slog.Level to string
func stringFromSlogLevel(level slog.Level) string {
	for k, v := range logLevels {
		if v == level {
			return k
		}
	}
	return "Debug" // default if not found
}

// writeConfigToFile writes a json representation of the config to a file
func writeConfigToFile(config *Config) error {
	// Marshal backups array to JSON
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	// Write JSON data to file
	err = os.WriteFile(fmt.Sprintf("%s/%s", config.DataDirectory, "config.json"), data, 0644)
	if err != nil {
		return err
	}

	return nil
}

// readConfigFromFile reads a json config and returns it as a Config struct
func readConfigFromFile(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// getIngressEntry returns the hassio ingress path for the addon
func getIngressEntry(token string) string {
	bearer := "Bearer " + token

	req, err := http.NewRequest("GET", "http://supervisor/addons/self/info", nil)
	if err != nil {
		fmt.Println(err)
	}

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
	}

	// Unmarshal JSON into the struct
	var response hassio.HassioResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error decoding JSON:", err)
	}

	// Access the IngressPath
	ingressEntry := response.Data.IngressEntry

	return ingressEntry
}
