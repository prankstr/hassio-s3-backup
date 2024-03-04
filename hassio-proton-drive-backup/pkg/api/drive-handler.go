package api

import (
	"net/http"

	"hassio-proton-drive-backup/models"
)

// ProtonDriveHandler handles requests to the /ProtonDrive and /bye endpoints
type DriveHandler struct {
	Drive models.Drive
}

// NewDriveHandler initializes and returns a new ProtonDriveHandler
func NewDriveHandler(drive models.Drive) DriveHandler {
	return DriveHandler{
		Drive: drive,
	}
}

// handleProtonDrive handles requests to /ProtonDrive
func (h *DriveHandler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for preflight OPTIONS request
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:9123")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers for main request
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:9123")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Access h.ProtonDriveService for /ProtonDrive logic
	res, _ := h.Drive.About()

	// Set the Content-Type header to indicate JSON
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON response
	w.Write(res)
}
