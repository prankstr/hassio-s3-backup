package api

import (
	"encoding/json"
	"fmt"
	"hassio-proton-drive-backup/models"
	"hassio-proton-drive-backup/pkg/services"
	"net/http"
)

// BackupHandler handles requests to the /ProtonDrive and /bye endpoints
type BackupHandler struct {
	backupService *services.BackupService
}

// NewBackupHandler initializes and returns a new BackupHandler
func NewBackupHandler(backupService *services.BackupService) BackupHandler {
	return BackupHandler{
		backupService: backupService,
	}
}

func (h *BackupHandler) HandleListBackups(w http.ResponseWriter, r *http.Request) {
	backups := h.backupService.ListBackups()

	// Marshal the backups into JSON
	json, err := json.Marshal(backups)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	// Write the JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

// handleProtonDrive handles requests to /ProtonDrive
func (h *BackupHandler) HandleBackupRequest(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	requestBody, err := parseRequest(r)
	if err != nil {
		handleError(w, err, http.StatusBadRequest)
		return
	}

	go func() {
		err := h.backupService.PerformBackup(requestBody.Name)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError)
			return
		} else {
			// Notify user of success
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

func (h *BackupHandler) HandleDeleteBackupRequest(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	requestBody, err := parseRequest(r)
	if err != nil {
		handleError(w, err, http.StatusBadRequest)
		return
	}

	err = h.backupService.DeleteBackup(requestBody.ID)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *BackupHandler) HandleRestoreBackupRequest(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	requestBody, err := parseRequest(r)
	if err != nil {
		handleError(w, err, http.StatusBadRequest)
		return
	}

	err = h.backupService.RestoreBackup(requestBody.Slug)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// parseRequest decodes the JSON request body into a BackupRequest struct
func parseRequest(r *http.Request) (*models.BackupRequest, error) {
	var requestBody models.BackupRequest

	err := json.NewDecoder(r.Body).Decode(&requestBody)

	return &requestBody, err
}

// handleError handles errors by logging them and writing an error response to the client
func handleError(w http.ResponseWriter, err error, statusCode int) {
	fmt.Println("Error:", err)
	http.Error(w, err.Error(), statusCode)
}
