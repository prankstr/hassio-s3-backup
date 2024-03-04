package clients

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hassio-proton-drive-backup/models"
	"io"
	"net/http"
	"time"
)

// hassioClient is the concrete implementation of HassioClient
type HassioApiClient struct {
	Token string
	URL   string
}

// NewHassioClient initializes and returns a new HassioClient
func NewHassioApiClient(token string) HassioApiClient {
	return HassioApiClient{
		Token: token,
		URL:   "http://supervisor",
	}
}

// ListBackups queries hassio for all current backups
func (c *HassioApiClient) GetBackup(slug string) (*models.HassBackup, error) {
	// API endpoint to list all backups
	url := fmt.Sprintf("http://supervisor/backups/%s/info", slug)

	// Create the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	// Perform the request
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the response
	// Define a struct to hold the JSON response
	var backupResponse struct {
		Result string            `json:"result"`
		Data   models.HassBackup `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&backupResponse); err != nil {
		fmt.Println("Error decoding response body:", err)
		return nil, err
	}

	// Check if the response is successful
	if backupResponse.Result != "ok" {
		return nil, fmt.Errorf("failed to get backup: %s", backupResponse.Result)
	}

	return &backupResponse.Data, nil
}

// ListBackups queries hassio for all current backups
func (c *HassioApiClient) ListBackups() ([]*models.HassBackup, error) {
	// API endpoint to list all backups
	url := "http://supervisor/backups"

	// Create the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the response
	var backupResponse models.HassBackupResponse
	if err := json.NewDecoder(resp.Body).Decode(&backupResponse); err != nil {
		fmt.Println("Error decoding response body:", err)
		return nil, err
	}

	return backupResponse.Data.Backups, nil
}

// BackupFull handles requests to /ProtonDrive
func (c *HassioApiClient) BackupFull(name string) (string, error) {
	jsonBody := []byte(fmt.Sprintf(`{"name": "%s"}`, name))
	bodyReader := bytes.NewReader(jsonBody)

	req, err := http.NewRequest(http.MethodPost, "http://supervisor/backups/new/full", bodyReader)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	client := http.Client{Timeout: 120 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var response models.HassioResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if response.Result == "error" {
		return "", errors.New(response.Message)
	}

	return response.Data.Slug, nil
}

func (c *HassioApiClient) DeleteBackup(slug string) error {
	url := fmt.Sprintf("http://supervisor/backups/%s", slug)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *HassioApiClient) RestoreBackup(slug string) error {
	url := fmt.Sprintf("http://supervisor/backups/%s/restore/full", slug)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
