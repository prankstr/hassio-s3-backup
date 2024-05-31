package server

import (
	"hassio-proton-drive-backup/api/server/backup"
	"hassio-proton-drive-backup/api/server/config"
	"hassio-proton-drive-backup/api/server/storage"
	"hassio-proton-drive-backup/internal"
	"net/http"
)

// Api struct holds dependencies for the API
type Server struct {
	Router *http.ServeMux
}

// New initializes and returns a new server
func New(services *internal.Services) (*Server, error) {
	mux := http.NewServeMux()

	// Register routes
	backup.RegisterBackupRoutes(mux, services)
	config.RegisterConfigRoutes(mux, services)
	storage.RegisterStorageRoutes(mux, services)

	return &Server{
		Router: mux,
	}, nil
}
