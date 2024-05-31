package config

import (
	"hassio-proton-drive-backup/internal"
	"net/http"
)

// NewConfigRouter creates a new ConfigRouter.
func RegisterConfigRoutes(mux *http.ServeMux, services *internal.Services) {
	h := newConfigHandler(services.ConfigService)

	mux.HandleFunc("GET /api/config", h.handleGetConfig)
	mux.HandleFunc("POST /api/config/update", h.handleUpdateConfig)
}
