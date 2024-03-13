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

	configHandler := NewConfigHandler(configService)

	// Define routes
	router.HandleFunc("GET /api/backups", backupHandler.HandleListBackups)
	router.HandleFunc("GET /api/backups/{id}/download", backupHandler.HandleDownloadBackupRequest)
	router.HandleFunc("GET /api/backups/timer", backupHandler.HandleTimerRequest)
	router.HandleFunc("GET /api/backups/reset", backupHandler.HandleResetBackupsRequest)
	router.HandleFunc("POST /api/backups/new/full", backupHandler.HandleBackupRequest)
	router.HandleFunc("POST /api/backups/{id}/pin", backupHandler.HandlePinBackupRequest)
	router.HandleFunc("POST /api/backups/{id}/unpin", backupHandler.HandleUnpinBackupRequest)
	router.HandleFunc("POST /api/backups/{id}/restore", backupHandler.HandleRestoreBackupRequest)
	router.HandleFunc("DELETE /api/backups/{id}", backupHandler.HandleDeleteBackupRequest)

	router.HandleFunc("GET /api/config", configHandler.HandleGetConfig)
	router.HandleFunc("POST /api/config/update", configHandler.HandleUpdateConfig)

	router.HandleFunc("GET /api/drive/about", driveHandler.HandleAbout)

	return &Api{
		Router: router,
	}, nil
}
