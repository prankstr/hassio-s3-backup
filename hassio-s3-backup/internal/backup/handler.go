package backup

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// backupHandler is a router for backup-related routes.
type backupHandler struct {
	backupService *Service
}

// backupRequest represents the expected JSON structure for backup requests.
type backupRequest struct {
	Name string `json:"name"`
}

// newBackupHandler creates and returns a new backupHandler instance.
func newBackupHandler(bs *Service) *backupHandler {
	return &backupHandler{
		backupService: bs,
	}
}

// handleListBackups handles the listing of backups.
func (h *backupHandler) handleListBackups(w http.ResponseWriter, r *http.Request) {
	backups := h.backupService.ListBackups()

	// Marshal the backups into JSON
	jsonData, err := json.Marshal(backups)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	// Write the JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// handleTimerRequest handles requests to get the time until the next backup.
func (h *backupHandler) handleTimerRequest(w http.ResponseWriter, r *http.Request) {
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

// handleBackupRequest handles requests to perform a backup.
func (h *backupHandler) handleBackupRequest(w http.ResponseWriter, r *http.Request) {
	var requestBody backupRequest

	// Decode the JSON request body into the backupRequest struct.
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if h.backupService.NameExists(requestBody.Name) {
		handleError(w, fmt.Errorf("a backup with the name \"%s\" already exists", requestBody.Name), http.StatusBadRequest)
		return
	}

	go func() {
		slog.Info("backup request received", "name", requestBody.Name)
		err := h.backupService.PerformBackup(requestBody.Name)
		if err != nil {
			slog.Error("error performing backup", "error", err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

// handleDeleteBackupRequest handles requests to delete a backup.
func (h *backupHandler) handleDeleteBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	err := h.backupService.DeleteBackup(id)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleRestoreBackupRequest handles requests to restore a backup.
func (h *backupHandler) handleRestoreBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	err := h.backupService.RestoreBackup(id)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleDownloadBackupRequest handles requests to download a backup.
func (h *backupHandler) handleDownloadBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	err := h.backupService.DownloadBackup(id)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handlePinBackupRequest handles requests to pin a backup.
func (h *backupHandler) handlePinBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	slog.Debug("received request to pin backup", "id", id)

	err := h.backupService.PinBackup(id)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleUnpinBackupRequest handles requests to unpin a backup.
func (h *backupHandler) handleUnpinBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	slog.Debug("received request to unpin backup", "id", id)

	err := h.backupService.UnpinBackup(id)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleResetBackupsRequest handles requests to reset backups.
func (h *backupHandler) handleResetBackupsRequest(w http.ResponseWriter, r *http.Request) {
	err := h.backupService.ResetBackups()
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleError handles errors by logging them and writing an error response to the client.
func handleError(w http.ResponseWriter, err error, statusCode int) {
	slog.Error("error handling request", "error", err, "status_code", statusCode)
	http.Error(w, err.Error(), statusCode)
}
