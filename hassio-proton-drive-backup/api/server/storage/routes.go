package storage

import (
	"hassio-proton-drive-backup/internal"
	"net/http"
)

// NewStorageRouter creates a new StorageRouter.
func RegisterStorageRoutes(mux *http.ServeMux, services *internal.Services) {
	h := NewStorageHandler(services.StorageService)
	mux.HandleFunc("GET /api/storage/about", h.HandleAbout)
}
