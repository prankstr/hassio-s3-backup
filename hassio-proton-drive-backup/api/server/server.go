package api

import (
	"fmt"
	"hassio-proton-drive-backup/models"
	"hassio-proton-drive-backup/pkg/backends"
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

	var err error
	var storageService models.StorageService
	switch config.StorageBackend {
	case "Storj":
		storageService, err = backends.NewStorjService(configService)
		if err != nil {
			return nil, err
		}
	case "Proton Drive":
		storageService, err = backends.NewProtonDriveService(configService)
		if err != nil {
			return nil, err
		}
	case "S3":
		storageService, err = backends.NewS3Service(configService)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown storage backend %s - check configuration", config.StorageBackend)
	}

	storageHandler := NewStorageHandler(storageService)

	backupService := services.NewBackupService(hassioApiClient, storageService, configService)
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

	router.HandleFunc("GET /api/storage/about", storageHandler.HandleAbout)

	return &Api{
		Router: router,
	}, nil
}
