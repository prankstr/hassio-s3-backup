package config

import (
	"encoding/json"
	"net/http"
)

// ConfigHandler is a router for config-related routes.
type configHandler struct {
	configService *Service // Service to handle configuration operations
}

// newConfigHandler creates and returns a new configHandler instance.
func newConfigHandler(cs *Service) *configHandler {
	return &configHandler{
		configService: cs,
	}
}

// handleGetConfig handles GET requests to retrieve the current configuration.
func (h *configHandler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	// Retrieve the current configuration
	conf := h.configService.GetConfig()

	// Prepare the response with the configuration options
	responseConfig := Options{
		BackupNameFormat: conf.BackupNameFormat,
		BackupInterval:   conf.BackupInterval,
		BackupsInHA:      conf.BackupsInHA,
		BackupsInS3:      conf.BackupsInS3,
	}

	// Marshal the responseConfig struct to JSON
	res, _ := json.Marshal(responseConfig)

	// Set the content type of the response to application/json
	w.Header().Set("Content-Type", "application/json")
	// Write the JSON response
	w.Write(res)
}

// handleUpdateConfig handles POST requests to update the configuration.
func (h *configHandler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	// Define a variable to hold the request body
	var requestBody Options

	// Decode the JSON request body into the requestBody struct
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		// If there's an error decoding the request body, return an internal server error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update the configuration using the config service
	err = h.configService.UpdateConfigFromAPI(requestBody)
	if err != nil {
		// If there's an error updating the configuration, return an internal server error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return a 200 OK status to indicate the configuration was updated successfully
	w.WriteHeader(http.StatusOK)
}
