package backup

import (
	"net/http"
)

// RegisterBackupRoutes registers routes for backup endpoints
func RegisterBackupRoutes(mux *http.ServeMux, bs *Service) {
	h := newBackupHandler(bs)

	mux.HandleFunc("GET /api/backups", h.handleListBackups)
	mux.HandleFunc("GET /api/backups/{id}/download", h.handleDownloadBackupRequest)
	mux.HandleFunc("GET /api/backups/timer", h.handleTimerRequest)
	mux.HandleFunc("POST /api/backups/reset", h.handleResetBackupsRequest)
	mux.HandleFunc("POST /api/backups/new/full", h.handleBackupRequest)
	mux.HandleFunc("POST /api/backups/{id}/pin", h.handlePinBackupRequest)
	mux.HandleFunc("POST /api/backups/{id}/unpin", h.handleUnpinBackupRequest)
	mux.HandleFunc("POST /api/backups/{id}/restore", h.handleRestoreBackupRequest)
	mux.HandleFunc("DELETE /api/backups/{id}", h.handleDeleteBackupRequest)
}
