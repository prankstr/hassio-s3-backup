package api

import (
	"hassio-proton-drive-backup/pkg/clients"
	"hassio-proton-drive-backup/pkg/services"
	"hassio-proton-drive-backup/utils/httpdebug"
	"net/http"
)

// Proxy struct holds dependencies for the Proxy
type Api struct {
	Router *http.ServeMux
}

// NewAPI initializes and returns a new API
func NewAPI(configService *services.ConfigService) (*Api, error) {
	router := http.NewServeMux()

	config := configService.GetConfig()
	if config.Debug {
		router.HandleFunc("/", httpdebug.Handler)
	}

	hassioApiClient := clients.NewHassioApiClient(config.SupervisorToken)

	driveService, err := services.NewProtonDriveService(configService)
	if err != nil {
		return nil, err
	}
	driveHandler := NewDriveHandler(&driveService)

	backupService := services.NewBackupService(hassioApiClient, &driveService, configService)
	backupHandler := NewBackupHandler(backupService)

	ConfigHandler := NewConfigHandler(configService)

	// Define routes
	//router.Handle("/api/backups", http.HandlerFunc(backupHandler.HandleListBackupRequest))
	router.Handle("/api/backups/new/full", http.HandlerFunc(backupHandler.HandleBackupRequest))
	router.Handle("/api/backups/delete", http.HandlerFunc(backupHandler.HandleDeleteBackupRequest))
	router.Handle("/api/backups/restore", http.HandlerFunc(backupHandler.HandleRestoreBackupRequest))
	router.Handle("/api/backups", http.HandlerFunc(backupHandler.HandleListBackups))
	router.Handle("/api/config", http.HandlerFunc(ConfigHandler.HandleGetConfig))
	router.Handle("/api/config/update", http.HandlerFunc(ConfigHandler.HandleUpdateConfig))

	router.Handle("/api/drive/about", http.HandlerFunc(driveHandler.HandleAbout))
	return &Api{
		Router: router,
	}, nil
}
