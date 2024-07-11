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

func newBackupHandler(bs *Service) *backupHandler {
	return &backupHandler{
		backupService: bs,
	}
}

func (h backupHandler) handleListBackups(w http.ResponseWriter, r *http.Request) {
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

func (h *backupHandler) handleBackupRequest(w http.ResponseWriter, r *http.Request) {
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

func (h *backupHandler) handleDeleteBackupRequest(w http.ResponseWriter, r *http.Request) {
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

func (h *backupHandler) handleRestoreBackupRequest(w http.ResponseWriter, r *http.Request) {
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

func (h *backupHandler) handleDownloadBackupRequest(w http.ResponseWriter, r *http.Request) {
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

func (h *backupHandler) handlePinBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	slog.Debug("received request to pin backup", "id", id)

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

func (h *backupHandler) handleUnpinBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	slog.Debug("received request to unpin backup", "id", id)

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

func (h *backupHandler) handleResetBackupsRequest(w http.ResponseWriter, r *http.Request) {
	err := h.backupService.ResetBackups()
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// parseRequest decodes the JSON request body into a BackupRequest struct
func parseRequest(r *http.Request) (*Request, error) {
	var requestBody Request

	err := json.NewDecoder(r.Body).Decode(&requestBody)

	return &requestBody, err
}

// handleError handles errors by logging them and writing an error response to the client
func handleError(w http.ResponseWriter, err error, statusCode int) {
	slog.Error("error handling request", "error", err, "status_code", statusCode)
	http.Error(w, err.Error(), statusCode)
}
