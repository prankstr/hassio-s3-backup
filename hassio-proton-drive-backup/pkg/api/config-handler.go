package api

import (
	"encoding/json"
	"hassio-proton-drive-backup/models"
	"hassio-proton-drive-backup/pkg/services"
	"net/http"
)

// ConfigHandler handles requests to the /ProtonDrive and /bye endpoints
type ConfigHandler struct {
	cs *services.ConfigService
}

// NewDriveHandler initializes and returns a new ProtonDriveHandler
func NewConfigHandler(configService *services.ConfigService) ConfigHandler {
	return ConfigHandler{
		cs: configService,
	}
}

// handleProtonDrive handles requests to /ProtonDrive
func (h *ConfigHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	config := h.cs.GetConfig()

	responseConfig := models.ConfigUpdate{
		BackupInterval: config.BackupInterval,
		BackupsToKeep:  config.BackupsToKeep,
	}

	// Access h.ProtonDriveService for /ProtonDrive logic
	res, _ := json.Marshal(responseConfig)

	// Set the Content-Type header to indicate JSON
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON response
	w.Write(res)
}

// handleProtonDrive handles requests to /ProtonDrive
func (h *ConfigHandler) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var requestBody models.ConfigUpdate

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
