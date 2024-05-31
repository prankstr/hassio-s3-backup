package config

import (
	"encoding/json"
	"hassio-proton-drive-backup/internal/config"
	"net/http"
)

// ConfigHandler is a router for config-related routes.
type ConfigHandler struct {
	configService *config.Service
}

func NewConfigHandler(configService *config.Service) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
	}
}

func (h *ConfigHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
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

func (h *ConfigHandler) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
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
