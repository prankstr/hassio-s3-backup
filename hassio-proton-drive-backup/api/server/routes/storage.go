package api

import (
	"hassio-proton-drive-backup/models"
	"net/http"
)

// DriveHandler is a struct to handle requests to concering the drive
type StorageHandler struct {
	StorageService models.StorageService
}

// NewDriveHandler initializes and returns a new ProtonDriveHandler
func NewStorageHandler(storageService models.StorageService) StorageHandler {
	return StorageHandler{
		StorageService: storageService,
	}
}

// HandleAbout handles requests to /drive/about, returning information about the drive
func (h *StorageHandler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	res, _ := h.StorageService.About()

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}
