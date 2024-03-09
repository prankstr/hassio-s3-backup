package api

import (
	"net/http"

	"hassio-proton-drive-backup/models"
)

// DriveHandler is a struct to handle requests to concering the drive
type DriveHandler struct {
	Drive models.Drive
}

// NewDriveHandler initializes and returns a new ProtonDriveHandler
func NewDriveHandler(drive models.Drive) DriveHandler {
	return DriveHandler{
		Drive: drive,
	}
}

// HandleAbout handles requests to /drive/about, returning information about the drive
func (h *DriveHandler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	res, _ := h.Drive.About()

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}
