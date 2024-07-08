package config

import (
	"encoding/json"
	"fmt"
	"hassio-proton-drive-backup/internal/hassio"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Config is a struct to represent the configuration of the addon
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

type S3Options struct {
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Endpoint        string
}

// Service is a struct to handle the application configuration
type Service struct {
	Config           *Options
	ConfigChangeChan chan *Options
}

// logLevels is a map of string to slog.Level
var logLevels map[string]slog.Level = map[string]slog.Level{
	"Error": slog.LevelError,
	"Warn":  slog.LevelWarn,
	"Info":  slog.LevelInfo,
	"Debug": slog.LevelDebug,
}

// New returns a new Config struct
func NewConfigService() *Service {
	// Read configuration from file
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
	config.BackupsInS3 = getEnvOrDefaultInt("BACKUPS_IN_STORAGE", config.BackupsInS3, 0)
	config.BackupInterval = getEnvOrDefaultInt("BACKUP_INTERVAL", config.BackupInterval, 3)

	// S3 Config
	config.S3.AccessKeyID = getEnvOrDefault("S3_ACCESS_KEY_ID", config.S3.AccessKeyID, "")
	config.S3.SecretAccessKey = getEnvOrDefault("S3_SECRET_ACCESS_KEY", config.S3.SecretAccessKey, "")
	config.S3.Bucket = getEnvOrDefault("S3_BUCKET_NAME", config.S3.Bucket, "")
	config.S3.Endpoint = getEnvOrDefault("S3_ENDPOINT", config.S3.Endpoint, "")

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

// GetBackupDirectory returns the directory where backups are stored
func (s *Service) GetBackupInterval() time.Duration {
	return (time.Duration(s.Config.BackupInterval) * time.Hour) * 24
}

func (s *Service) GetS3Bucket() string {
	return s.Config.S3.Bucket
}

func (s *Service) GetS3Endpoint() string {
	return s.Config.S3.Endpoint
}

func (s *Service) GetS3AccessKeyID() string {
	return s.Config.S3.AccessKeyID
}

func (s *Service) GetS3SecretAccessKey() string {
	return s.Config.S3.SecretAccessKey
}

// GetBackupNameFormat returns the format to use for the backup name
func (s *Service) GetBackupNameFormat() string {
	return s.Config.BackupNameFormat
}

// GetBackupsToKeep returns the number of backups to keep
func (s *Service) GetBackupsInHA() int {
	return s.Config.BackupsInHA
}

// GetBackupsOnDrive returns the number of backups to keep on the drive
func (s *Service) GetBackupsInS3() int {
	return s.Config.BackupsInS3
}

// SetBackupsToKeep sets the number of backups to keep
func (s *Service) SetBackupInterval(interval int) {
	s.Config.BackupInterval = interval

	writeConfigToFile(s.Config)
}

// SetBackupsToKeep sets the number of backups to keep
func (s *Service) UpdateConfigFromAPI(configRequest Options) error {
	s.Config.BackupNameFormat = configRequest.BackupNameFormat
	s.Config.BackupInterval = configRequest.BackupInterval
	s.Config.BackupsInHA = configRequest.BackupsInHA
	s.Config.BackupsInS3 = configRequest.BackupsInS3

	s.NotifyConfigChange(s.Config)

	writeConfigToFile(s.Config)
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
func writeConfigToFile(config *Options) error {
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
	var response hassio.Response
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error decoding JSON:", err)
	}

	// Access the IngressPath
	ingressEntry := response.Data.IngressEntry

	return ingressEntry
}
