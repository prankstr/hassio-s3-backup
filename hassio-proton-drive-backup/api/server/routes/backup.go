package api

import (
	"encoding/json"
	"fmt"
	"hassio-proton-drive-backup/models"
	"hassio-proton-drive-backup/pkg/services"
	"net/http"
)

// BackupHandler is a struct to handle requests to concering backups
type BackupHandler struct {
	backupService *services.BackupService
}

// NewBackupHandler initializes and returns a new BackupHandler
func NewBackupHandler(backupService *services.BackupService) BackupHandler {
	return BackupHandler{
		backupService: backupService,
	}
}

// HandleListBackups handles requests to /backups, returning a list of backups
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

// handleTimer handles requests to /timer, returning the time until the next backup
func (h *BackupHandler) HandleTimerRequest(w http.ResponseWriter, r *http.Request) {
	milliseconds := h.backupService.TimeUntilNextBackup()

	response := struct {
		Milliseconds int64 `json:"milliseconds"`
	}{
		Milliseconds: milliseconds,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)
}

// handleBackup handles requests to /backup, createing a new backup
func (h *BackupHandler) HandleBackupRequest(w http.ResponseWriter, r *http.Request) {
	requestBody, err := parseRequest(r)
	if err != nil {
		handleError(w, err, http.StatusBadRequest)
		return
	}

	if h.backupService.NameExists(requestBody.Name) {
		handleError(w, fmt.Errorf("a backup with the name \"%s\" already exists", requestBody.Name), http.StatusBadRequest)
		return
	}

	go func() {
		err := h.backupService.PerformBackup(requestBody.Name)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError)
			return
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

// handleDeleteBackup handles requests to /delete, deleting a backup
func (h *BackupHandler) HandleDeleteBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := h.backupService.DeleteBackup(id)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleRestoreBackup handles requests to /restore, restoring a backup
func (h *BackupHandler) HandleRestoreBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := h.backupService.RestoreBackup(id)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleRestoreBackup handles requests to /download, downloading a backup
func (h *BackupHandler) HandleDownloadBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := h.backupService.DownloadBackup(id)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handlePinBackup handles requests to /pin, pinning a backup
func (h *BackupHandler) HandlePinBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := h.backupService.PinBackup(id)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleUnpinBackup handles requests to /unpin, unpinning a backup
func (h *BackupHandler) HandleUnpinBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := h.backupService.UnpinBackup(id)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleResetBackups handles requests to /reset, resetting all backups
func (h *BackupHandler) HandleResetBackupsRequest(w http.ResponseWriter, r *http.Request) {
	err := h.backupService.ResetBackups()
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
