package storage

import (
	"hassio-proton-drive-backup/api/server/router"
	"hassio-proton-drive-backup/internal/storage"
	"net/http"
)

// StorageRouter is a router for storage-related routes.
type StorageRouter struct {
	*router.BaseRouter
	storageService storage.Service
}

// NewStorageRouter creates a new StorageRouter.
func NewStorageRouter(storageService storage.Service) *StorageRouter {
	sr := &StorageRouter{
		BaseRouter:     &router.BaseRouter{},
		storageService: storageService,
	}

	sr.AddRoute("GET", "/api/storage/about", sr.handleAbout)

	return sr
}

// handleAbout handles the /api/storage/about endpoint.
func (sr *StorageRouter) handleAbout(w http.ResponseWriter, r *http.Request) {
	res, _ := sr.storageService.About()

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}
