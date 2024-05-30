package server

import (
	apiBackup "hassio-proton-drive-backup/api/server/router/backup"
	apiConfig "hassio-proton-drive-backup/api/server/router/config"
	apiStorage "hassio-proton-drive-backup/api/server/router/storage"
	"hassio-proton-drive-backup/internal/config"
	"hassio-proton-drive-backup/internal/storage"
	"net/http"
)

// Api struct holds dependencies for the API
type Server struct {
	Router *http.ServeMux
}

// NewServer initializes and returns a new API
func NewServer(configService *config.Service, storageService storage.Service) (*Server, error) {
	router := http.NewServeMux()

	// Initialize and register routes for each module
	backupRouter := apiBackup.NewBackupRouter(storageService, configService)
	configRouter := apiConfig.NewConfigRouter(configService)
	storageRouter := apiStorage.NewStorageRouter(storageService)

	backupRouter.RegisterRoutes(router)
	configRouter.RegisterRoutes(router)
	storageRouter.RegisterRoutes(router)

	return &Server{
		Router: router,
	}, nil
}
