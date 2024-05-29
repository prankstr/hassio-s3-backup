package api

import (
	"encoding/json"
	"hassio-proton-drive-backup/models"
	"hassio-proton-drive-backup/pkg/services"
	"net/http"
)

// ConfigHandler is a struct to handle requests to concering config
type ConfigHandler struct {
	cs *services.ConfigService
}

// NewConfigHandler initializes and returns a new ConfigHandler
func NewConfigHandler(configService *services.ConfigService) ConfigHandler {
	return ConfigHandler{
		cs: configService,
	}
}

// HandleGetConfig handles requests to /config, returning the current config
func (h *ConfigHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	config := h.cs.GetConfig()

	responseConfig := models.Config{
		StorageBackend:   config.StorageBackend,
		BackupNameFormat: config.BackupNameFormat,
		BackupInterval:   config.BackupInterval,
		BackupsInHA:      config.BackupsInHA,
		BackupsInStorage: config.BackupsInStorage,
	}

	res, _ := json.Marshal(responseConfig)

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

// HandleUpdateConfig handles requests to /config/update, updating the config
func (h *ConfigHandler) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var requestBody models.Config

	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	err = h.cs.UpdateConfigFromAPI(requestBody)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
