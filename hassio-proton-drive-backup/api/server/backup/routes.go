package backup

import (
	"hassio-proton-drive-backup/internal"
	"net/http"
)

// NewBackupRouter creates a new BackupHandler.
func RegisterBackupRoutes(mux *http.ServeMux, services *internal.Services) {
	h := NewBackupHandler(services.StorageService, services.ConfigService)

	mux.HandleFunc("GET /api/backups", h.HandleListBackups)
	mux.HandleFunc("GET /api/backups/{id}/download", h.HandleDownloadBackupRequest)
	mux.HandleFunc("GET /api/backups/timer", h.HandleTimerRequest)
	mux.HandleFunc("GET /api/backups/reset", h.HandleResetBackupsRequest)
	mux.HandleFunc("POST /api/backups/new/full", h.HandleBackupRequest)
	mux.HandleFunc("POST /api/backups/{id}/pin", h.HandlePinBackupRequest)
	mux.HandleFunc("POST /api/backups/{id}/unpin", h.HandleUnpinBackupRequest)
	mux.HandleFunc("POST /api/backups/{id}/restore", h.HandleRestoreBackupRequest)
	mux.HandleFunc("DELETE /api/backups/{id}", h.HandleDeleteBackupRequest)
}
