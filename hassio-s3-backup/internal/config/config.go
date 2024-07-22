package config

import (
	"encoding/json"
	"fmt"
	"hassio-proton-drive-backup/internal/hassio"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Options represents the addon options
type Options struct {
	Timezone         *time.Location
	S3               S3Options
	DataDirectory    string
	SupervisorToken  string
	BackupNameFormat string `json:"backupNameFormat"`
	IngressPath      string
	Hostname         string
	LogLevel         slog.Level
	BackupInterval   int `json:"backupInterval"`
	BackupsInHA      int `json:"backupsInHA"`
	BackupsInS3      int `json:"backupsInS3"`
	Debug            bool
}

// S3Options represents the S3 options
type S3Options struct {
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Endpoint        string
}

// Service represents the config service
type Service struct {
	Config           *Options
	ConfigChangeChan chan *Options
}

// logLevels maps string to slog.Level
var logLevels map[string]slog.Level = map[string]slog.Level{
	"Error": slog.LevelError,
	"Warn":  slog.LevelWarn,
	"Info":  slog.LevelInfo,
	"Debug": slog.LevelDebug,
}

// NewConfigService returns a new ConfigService
func NewConfigService() *Service {
	// Read saved configuration from file
	config, err := readConfigFromFile("/data/config.json")
	if err != nil {
		// Handle error. This could be logging the error and continuing with defaults
		slog.Error("Error reading config file: ", err)
		config = &Options{} // Initialize with an empty config
	}

	// Set defaults or override with environment variables if they are set
	config.Hostname = getEnvOrDefault("HOSTNAME", config.Hostname, "localhost:9101")
	config.SupervisorToken = getEnvOrDefault("SUPERVISOR_TOKEN", config.SupervisorToken, "")
	config.DataDirectory = getEnvOrDefault("DATA_DIRECTORY", config.DataDirectory, "/data")
	config.BackupsInHA = getEnvOrDefaultInt("BACKUPS_IN_HA", config.BackupsInHA, 0)
	config.BackupNameFormat = getEnvOrDefault("BACKUP_NAME_FORMAT", config.BackupNameFormat, "Full Backup {year}-{month}-{day} {hr24}:{min}:{sec}")
	config.BackupsInS3 = getEnvOrDefaultInt("BACKUPS_IN_S3", config.BackupsInS3, 0)
	config.BackupInterval = getEnvOrDefaultInt("BACKUP_INTERVAL", config.BackupInterval, 3)

	defaultTimezone := "UTC"
	timezoneStr := getEnvOrDefault("TZ", config.Timezone.String(), defaultTimezone)
	config.Timezone, err = time.LoadLocation(timezoneStr)
	if err != nil {
		slog.Error("Invalid time zone, defaulting to UTC: %v", err)
		config.Timezone, _ = time.LoadLocation(defaultTimezone)
	}

	logLevelStr := getEnvOrDefault("LOG_LEVEL", stringFromSlogLevel(config.LogLevel), "Debug")
	if level, exists := logLevels[logLevelStr]; exists {
		config.LogLevel = level
	} else {
		slog.Error("Invalid log level specified. Using default level.")
		config.LogLevel = logLevels["Info"]
	}

	debugStr := getEnvOrDefault("DEBUG", strconv.FormatBool(config.Debug), "true")
	debug, err := strconv.ParseBool(debugStr)
	if err == nil {
		config.Debug = debug
	} else {
		slog.Error("Cannot parse the DEBUG variable")
	}

	// S3 Config
	config.S3.AccessKeyID = getEnvOrDefault("S3_ACCESS_KEY_ID", config.S3.AccessKeyID, "")
	config.S3.SecretAccessKey = getEnvOrDefault("S3_SECRET_ACCESS_KEY", config.S3.SecretAccessKey, "")
	config.S3.Bucket = getEnvOrDefault("S3_BUCKET_NAME", config.S3.Bucket, "")
	config.S3.Endpoint = getEnvOrDefault("S3_ENDPOINT", config.S3.Endpoint, "")

	// Handle ingress entry
	ingressEntry, err := getIngressEntry(config.SupervisorToken)
	if err != nil {
		slog.Error("Error getting ingress entry: %v", err)
		ingressEntry = ""
	}
	config.IngressPath = ingressEntry

	// Write config to file
	err = writeConfigToFile(config)
	if err != nil {
		slog.Error("Error writing config to file: %v", err)
	}

	return &Service{
		Config:           config,
		ConfigChangeChan: make(chan *Options),
	}
}

// NotifyConfigChange sends a new config to the configChangeChan
func (s *Service) NotifyConfigChange(newConfig *Options) {
	slog.Info("Config updated, notifying")
	s.ConfigChangeChan <- newConfig
}

// GetConfig returns the current application configuration.
func (s *Service) GetConfig() *Options {
	return s.Config
}

// GetBackupInterval returns the backup interval in days
func (s *Service) GetBackupInterval() time.Duration {
	return (time.Duration(s.Config.BackupInterval) * time.Hour) * 24
}

// GetS3Bucket returns the S3 bucket name
func (s *Service) GetS3Bucket() string {
	return s.Config.S3.Bucket
}

// GetS3Endpoint returns the S3 endpoint
func (s *Service) GetS3Endpoint() string {
	return s.Config.S3.Endpoint
}

// GetS3AccessKeyID returns the S3 access key ID
func (s *Service) GetS3AccessKeyID() string {
	return s.Config.S3.AccessKeyID
}

// GetS3SecretAccessKey returns the S3 secret access key
func (s *Service) GetS3SecretAccessKey() string {
	return s.Config.S3.SecretAccessKey
}

// GetBackupNameFormat returns the format to use for the backup name
func (s *Service) GetBackupNameFormat() string {
	return s.Config.BackupNameFormat
}

// GetBackupsInHA returns the number of backups to keep in Home Assistant
func (s *Service) GetBackupsInHA() int {
	return s.Config.BackupsInHA
}

// GetBackupsInS3 returns the number of backups to keep in S3
func (s *Service) GetBackupsInS3() int {
	return s.Config.BackupsInS3
}

// SetBackupInterval sets the backup interval
func (s *Service) SetBackupInterval(interval int) {
	s.Config.BackupInterval = interval
	err := writeConfigToFile(s.Config)
	if err != nil {
		slog.Error("Error writing config to file: %v", err)
	}
}

// UpdateConfigFromAPI updates the configuration with the provided settings from an API request
func (s *Service) UpdateConfigFromAPI(configRequest Options) error {
	s.Config.BackupNameFormat = configRequest.BackupNameFormat
	s.Config.BackupInterval = configRequest.BackupInterval
	s.Config.BackupsInHA = configRequest.BackupsInHA
	s.Config.BackupsInS3 = configRequest.BackupsInS3

	s.NotifyConfigChange(s.Config)
	err := writeConfigToFile(s.Config)
	if err != nil {
		slog.Error("Error writing config to file: %v", err)
		return fmt.Errorf("failed to update config: %v", err)
	}
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

// Helper function to get environment variable as integer or return a default
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
func writeConfigToFile(config *Options) error {
	// Marshal config to JSON
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

// readConfigFromFile reads the config and returns it as an Options struct
func readConfigFromFile(filePath string) (*Options, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Options
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// getIngressEntry returns the hassio ingress path for the addon
func getIngressEntry(token string) (string, error) {
	bearer := "Bearer " + token

	req, err := http.NewRequest("GET", "http://supervisor/addons/self/info", nil)
	if err != nil {
		return "", err
	}

	// Add authorization header to the request
	req.Header.Add("Authorization", bearer)

	// Send request using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error on response: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// Unmarshal JSON into the struct
	var response hassio.Response
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("error decoding JSON: %v", err)
	}

	// Access the IngressPath
	return response.Data.IngressEntry, nil
}
