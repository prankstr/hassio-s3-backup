package config

import (
	"encoding/json"
	"hassio-proton-drive-backup/api/server/router"
	internalConfig "hassio-proton-drive-backup/internal/config"
	"net/http"
)

// ConfigRouter is a router for config-related routes.
type ConfigRouter struct {
	*router.BaseRouter
	configService *internalConfig.Service
}

// NewConfigRouter creates a new ConfigRouter.
func NewConfigRouter(configService *internalConfig.Service) *ConfigRouter {
	cr := &ConfigRouter{
		BaseRouter:    &router.BaseRouter{},
		configService: configService,
	}

	cr.AddRoute("GET", "/api/config", cr.handleGetConfig)
	cr.AddRoute("POST", "/api/config/update", cr.handleUpdateConfig)

	return cr
}

func (cr *ConfigRouter) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	conf := cr.configService.GetConfig()

	responseConfig := internalConfig.Options{
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

func (cr *ConfigRouter) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var requestBody internalConfig.Options

	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = cr.configService.UpdateConfigFromAPI(requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
