package server

import (
	"hassio-proton-drive-backup/api/server/backup"
	"hassio-proton-drive-backup/api/server/config"
	"hassio-proton-drive-backup/api/server/storage"
	"hassio-proton-drive-backup/internal"
	"net/http"
)

// Api struct holds dependencies for the API
type Api struct {
	Router *http.ServeMux
}

// NewServer initializes and returns a new API
func New(services *internal.Services) (*Api, error) {
	mux := http.NewServeMux()

	// Register routes
	backup.RegisterBackupRoutes(mux, services)
	config.RegisterConfigRoutes(mux, services)
	storage.RegisterStorageRoutes(mux, services)

	return &Api{
		Router: mux,
	}, nil
}
