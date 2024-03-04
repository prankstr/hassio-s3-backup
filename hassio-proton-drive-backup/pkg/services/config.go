package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"hassio-proton-drive-backup/models"
)

type ConfigService struct {
	config           *models.Config
	configChangeChan chan *models.Config
}

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
		config = &models.Config{} // Initialize with an empty config
	}

	// Set defaults or override with environment variables if they are set
	config.Hostname = getEnvOrDefault("HOSTNAME", config.Hostname, "localhost:9101")
	config.SupervisorToken = getEnvOrDefault("SUPERVISOR_TOKEN", config.SupervisorToken, "")
	config.BackupDirectory = getEnvOrDefault("BACKUP_DIRECTORY", config.BackupDirectory, "Home Assistant Backups")
	config.DataDirectory = getEnvOrDefault("DATA_DIRECTORY", config.DataDirectory, "/data")
	config.BackupsToKeep = getEnvOrDefaultInt("BACKUPS_TO_KEEP", config.BackupsToKeep, 4)
	config.BackupInterval = getEnvOrDefaultInt("BACKUP_INTERVAL", config.BackupInterval, 3)
	config.ProtonDriveUser = getEnvOrDefault("PROTON_DRIVE_USER", config.ProtonDriveUser, "")
	config.ProtonDrivePassword = getEnvOrDefault("PROTON_DRIVE_PASSWORD", config.ProtonDrivePassword, "")

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

	writeConfigToFile(config, "/data.config.json")

	return &ConfigService{
		config:           config,
		configChangeChan: make(chan *models.Config),
	}
}

func (cs *ConfigService) NotifyConfigChange(newConfig *models.Config) {
	slog.Info("Config updated, notifying")
	cs.configChangeChan <- newConfig
}

// GetConfig returns the current application configuration.
func (cs *ConfigService) GetConfig() *models.Config {
	return cs.config
}

func (cs *ConfigService) GetBackupInterval() time.Duration {
	return (time.Duration(cs.config.BackupInterval) * time.Hour) * 24
}

func (cs *ConfigService) GetBackupsToKeep() int {
	return cs.config.BackupsToKeep
}

func (cs *ConfigService) GetProtonDriveUser() string {
	return cs.config.ProtonDriveUser
}

func (cs *ConfigService) GetProtonDrivePassword() string {
	return cs.config.ProtonDrivePassword
}

func (cs *ConfigService) SetBackupInterval(interval int) {
	cs.config.BackupInterval = interval

	writeConfigToFile(cs.config, "/data/config.json")
}

func (cs *ConfigService) UpdateConfigFromAPI(configRequest models.ConfigUpdate) error {
	cs.config.BackupInterval = configRequest.BackupInterval
	cs.config.BackupsToKeep = configRequest.BackupsToKeep

	cs.NotifyConfigChange(cs.config)

	writeConfigToFile(cs.config, "/data.config.json")
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
func writeConfigToFile(config *models.Config, filePath string) error {
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
func readConfigFromFile(filePath string) (*models.Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config models.Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// getIngressEntry returns the hassio ingress path for the addon
func getIngressEntry(token string) string {
	var bearer = "Bearer " + token

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
	var response models.HassioResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error decoding JSON:", err)
	}

	// Access the IngressPath
	ingressEntry := response.Data.IngressEntry

	return ingressEntry
}
