package hassio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HassBackup represents the details of a backup in Home Assistant
type HassBackup struct {
	Date       time.Time     `json:"date"`
	Slug       string        `json:"slug"`
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Location   string        `json:"location"`
	Content    BackupContent `json:"content"`
	Size       float64       `json:"size"`
	Protected  bool          `json:"protected"`
	Compressed bool          `json:"compressed"`
}

// BackupContent represents the content of a backup
type BackupContent struct {
	Addons        []string `json:"addons"`
	Folders       []string `json:"folders"`
	HomeAssistant bool     `json:"homeassistant"`
}

// HassBackupResponse represents the response from Home Assistant
type HassBackupResponse struct {
	Result string `json:"result"`
	Data   struct {
		Backups []*HassBackup `json:"backups"`
	} `json:"data"`
}

// HassioResponseData represents the data in the response from Home Assistant
type HassioResponseData struct {
	Slug         string `json:"slug"`
	IngressEntry string `json:"ingress_entry"`
}

// HassioResponse represents the response from Home Assistant
type HassioResponse struct {
	Result  string `json:"result"`
	Message string `json:"message"`
	Data    HassioResponseData
}

// HassioApiClient is a client for the Hassio API
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

// GetBackup queries hassio for a specific backup
func (c *HassioApiClient) GetBackup(slug string) (*HassBackup, error) {
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
		Result string     `json:"result"`
		Data   HassBackup `json:"data"`
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

// ListBackups queries hassio for a list of all backups
func (c *HassioApiClient) ListBackups() ([]*HassBackup, error) {
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
	var backupResponse HassBackupResponse
	if err := json.NewDecoder(resp.Body).Decode(&backupResponse); err != nil {
		fmt.Println("Error decoding response body:", err)
		return nil, err
	}

	return backupResponse.Data.Backups, nil
}

// BackupFull requests a full backup from Home Assistant
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

	var response HassioResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if response.Result == "error" {
		return "", errors.New(response.Message)
	}

	return response.Data.Slug, nil
}

// DeleteBackup requests a backup to be deleted from Home Assistant
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

// RestoreBackup requests a backup to be restored in Home Assistant
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
