package config

import (
	"encoding/json"
	"hassio-proton-drive-backup/internal/config"
	"net/http"
)

// ConfigHandler is a router for config-related routes.
type configHandler struct {
	configService *config.Service
}

func newConfigHandler(configService *config.Service) *configHandler {
	return &configHandler{
		configService: configService,
	}
}

func (h *configHandler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	conf := h.configService.GetConfig()

	responseConfig := config.Options{
		StorageBackend:   conf.StorageBackend,
		BackupNameFormat: conf.BackupNameFormat,
		BackupInterval:   conf.BackupInterval,
		BackupsInHA:      conf.BackupsInHA,
		BackupsInStorage: conf.BackupsInStorage,
	}

	res, _ := json.Marshal(responseConfig)

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func (h *configHandler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var requestBody config.Options

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
