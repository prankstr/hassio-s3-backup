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
	/* 	if r.Method == http.MethodOptions {
	   		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:9123")
	   		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	   		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	   		w.WriteHeader(http.StatusOK)
	   		return
	   	}

	   	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:9123")
	   	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	   	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	*/
	res, _ := h.Drive.About()

	w.Header().Set("Content-Type", "application/json")

	w.Write(res)
}
