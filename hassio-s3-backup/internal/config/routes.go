package config

import (
	"net/http"
)

// RegisterConfigRoutes registers routes for config endpoints
func RegisterConfigRoutes(mux *http.ServeMux, cs *Service) {
	h := newConfigHandler(cs)

	mux.HandleFunc("GET /api/config", h.handleGetConfig)
	mux.HandleFunc("POST /api/config/update", h.handleUpdateConfig)
}
