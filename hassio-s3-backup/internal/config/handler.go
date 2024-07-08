package config

import (
	"encoding/json"
	"net/http"
)

// ConfigHandler is a router for config-related routes.
type configHandler struct {
	configService *Service
}

func newConfigHandler(cs *Service) *configHandler {
	return &configHandler{
		configService: cs,
	}
}

func (h *configHandler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	conf := h.configService.GetConfig()

	responseConfig := Options{
		BackupNameFormat: conf.BackupNameFormat,
		BackupInterval:   conf.BackupInterval,
		BackupsInHA:      conf.BackupsInHA,
		BackupsInS3:      conf.BackupsInS3,
	}

	res, _ := json.Marshal(responseConfig)

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func (h *configHandler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var requestBody Options

	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.configService.UpdateConfigFromAPI(requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
