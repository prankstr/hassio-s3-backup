package backup

import (
	"encoding/json"
	"fmt"
	"hassio-proton-drive-backup/api/server/httputils"
	"hassio-proton-drive-backup/api/server/router"
	internalBackup "hassio-proton-drive-backup/internal/backup"
	"hassio-proton-drive-backup/internal/config"
	"hassio-proton-drive-backup/internal/storage"
	"log/slog"
	"net/http"
)

// BackupRouter is a router for backup-related routes.
type BackupRouter struct {
	*router.BaseRouter
	backupService *internalBackup.Service
}

// NewBackupRouter creates a new BackupRouter.
func NewBackupRouter(storageService storage.Service, configService *config.Service) *BackupRouter {
	backupService := internalBackup.NewBackupService(storageService, configService)
	br := &BackupRouter{
		BaseRouter:    &router.BaseRouter{},
		backupService: backupService,
	}

	br.AddRoute("GET", "/api/backups", br.handleListBackups)
	br.AddRoute("GET", "/api/backups/{id}/download", br.handleDownloadBackupRequest)
	br.AddRoute("GET", "/api/backups/timer", br.handleTimerRequest)
	br.AddRoute("GET", "/api/backups/reset", br.handleResetBackupsRequest)
	br.AddRoute("POST", "/api/backups/new/full", br.handleBackupRequest)
	br.AddRoute("POST", "/api/backups/{id}/pin", br.handlePinBackupRequest)
	br.AddRoute("POST", "/api/backups/{id}/unpin", br.handleUnpinBackupRequest)
	br.AddRoute("POST", "/api/backups/{id}/restore", br.handleRestoreBackupRequest)
	br.AddRoute("DELETE", "/api/backups/{id}", br.handleDeleteBackupRequest)

	return br
}

func (br *BackupRouter) handleListBackups(w http.ResponseWriter, r *http.Request) {
	backups := br.backupService.ListBackups()

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

func (br *BackupRouter) handleTimerRequest(w http.ResponseWriter, r *http.Request) {
	milliseconds := br.backupService.TimeUntilNextBackup()

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

func (br *BackupRouter) handleBackupRequest(w http.ResponseWriter, r *http.Request) {
	requestBody, err := parseRequest(r)
	if err != nil {
		httputils.HandleError(w, err, http.StatusBadRequest)
		return
	}

	if br.backupService.NameExists(requestBody.Name) {
		httputils.HandleError(w, fmt.Errorf("a backup with the name \"%s\" already exists", requestBody.Name), http.StatusBadRequest)
		return
	}

	go func() {
		err := br.backupService.PerformBackup(requestBody.Name)
		if err != nil {
			httputils.HandleError(w, err, http.StatusInternalServerError)
			return
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

func (br *BackupRouter) handleDeleteBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			httputils.HandleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := br.backupService.DeleteBackup(id)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (br *BackupRouter) handleRestoreBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			httputils.HandleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := br.backupService.RestoreBackup(id)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (br *BackupRouter) handleDownloadBackupRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		requestBody, err := parseRequest(r)
		if err != nil {
			httputils.HandleError(w, err, http.StatusBadRequest)
			return
		}

		id = requestBody.ID
	}

	err := br.backupService.DownloadBackup(id)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (br *BackupRouter) handlePinBackupRequest(w http.ResponseWriter, r *http.Request) {
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

	err := br.backupService.PinBackup(id)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (br *BackupRouter) handleUnpinBackupRequest(w http.ResponseWriter, r *http.Request) {
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

	err := br.backupService.UnpinBackup(id)
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (br *BackupRouter) handleResetBackupsRequest(w http.ResponseWriter, r *http.Request) {
	err := br.backupService.ResetBackups()
	if err != nil {
		httputils.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// parseRequest decodes the JSON request body into a BackupRequest struct
func parseRequest(r *http.Request) (*internalBackup.Request, error) {
	var requestBody internalBackup.Request

	err := json.NewDecoder(r.Body).Decode(&requestBody)

	return &requestBody, err
}
