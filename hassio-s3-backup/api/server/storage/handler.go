package storage

import (
	"hassio-proton-drive-backup/internal/storage"
	"net/http"
)

// StorageRouter is a router for storage-related routes.
type storageHandler struct {
	storageService storage.Service
}

func newStorageHandler(storageService storage.Service) *storageHandler {
	return &storageHandler{
		storageService: storageService,
	}
}

// handleAbout handles the /api/storage/about endpoint.
func (h *storageHandler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	res, _ := h.storageService.About()

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}
