package storage

import (
	"hassio-proton-drive-backup/internal/storage"
	"net/http"
)

// StorageRouter is a router for storage-related routes.
type StorageHandler struct {
	storageService storage.Service
}

func NewStorageHandler(storageService storage.Service) *StorageHandler {
	return &StorageHandler{
		storageService: storageService,
	}
}

// handleAbout handles the /api/storage/about endpoint.
func (h *StorageHandler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	res, _ := h.storageService.About()

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}
