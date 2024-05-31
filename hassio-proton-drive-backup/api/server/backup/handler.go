package backup

import (
	"encoding/json"
	"fmt"
	"hassio-proton-drive-backup/api/server/httputils"
	"hassio-proton-drive-backup/internal/backup"
	"hassio-proton-drive-backup/internal/config"
	"hassio-proton-drive-backup/internal/storage"
	"log/slog"
	"net/http"
)

// BackupHandler is a router for backup-related routes.
type BackupHandler struct {
	backupService *backup.Service
}

func NewBackupHandler(storageService storage.Service, configService *config.Service) *BackupHandler {
	return &BackupHandler{
		backupService: backup.NewService(storageService, configService),
	}
}

func (h BackupHandler) HandleListBackups(w http.ResponseWriter, r *http.Request) {
	backups := h.backupService.ListBackups()

	// Marshal the backups into JSON
	json, err := json.Marshal(backups)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	// Write the JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func (h *BackupHandler) HandleTimerRequest(w http.ResponseWriter, r *http.Request) {
	milliseconds := h.backupService.TimeUntilNextBackup()

	response := struct {
		Milliseconds int64 `json:"milliseconds"`
	}{
		Milliseconds: milliseconds,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)
}

func (h *BackupHandler) HandleBackupRequest(w http.ResponseWriter, r *http.Request) {
	requestBody, err := parseRequest(r)
	if err != nil {
		httputils.HandleError(w, err, http.StatusBadRequest)
		return
	}

	if h.backupService.NameExists(requestBody.Name) {
		httputils.HandleError(w, fmt.Errorf("a backup with the name \"%s\" already exists", requestBody.Name), http.StatusBadRequest)
		return
	}

	go func() {
		err := h.backupService.PerformBackup(requestBody.Name)
		if err != nil {
			httputils.HandleError(w, err, http.StatusInternalServerError)
			return
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

func (h *BackupHandler) HandleDeleteBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			httputils.HandleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := h.backupService.DeleteBackup(id)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *BackupHandler) HandleRestoreBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			httputils.HandleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := h.backupService.RestoreBackup(id)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *BackupHandler) HandleDownloadBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			httputils.HandleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := h.backupService.DownloadBackup(id)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *BackupHandler) HandlePinBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	slog.Debug("received request to pin backup", "id", id)

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			httputils.HandleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := h.backupService.PinBackup(id)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *BackupHandler) HandleUnpinBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	slog.Debug("received request to unpin backup", "id", id)

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			httputils.HandleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := h.backupService.UnpinBackup(id)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *BackupHandler) HandleResetBackupsRequest(w http.ResponseWriter, r *http.Request) {
	err := h.backupService.ResetBackups()
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// parseRequest decodes the JSON request body into a BackupRequest struct
func parseRequest(r *http.Request) (*backup.Request, error) {
	var requestBody backup.Request

	err := json.NewDecoder(r.Body).Decode(&requestBody)

	return &requestBody, err
}
