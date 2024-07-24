package hassio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Backup represents the details of a backup in Home Assistant
type Backup struct {
	Date time.Time `json:"date"`
	Slug string    `json:"slug"`
	Name string    `json:"name"`
	Type string    `json:"type"`
	Size float64   `json:"size"`
}

// BaseResponse represents a generic response from Home Assistant
type BaseResponse struct {
	Data    interface{} `json:"data"`
	Result  string      `json:"result"`
	Message string      `json:"message"`
}

// BackupResponse represents the response from Home Assistant for a single backup
type BackupResponse struct {
	BaseResponse
	Data Backup `json:"data"`
}

// ListBackupsResponse represents the response from Home Assistant for listing backups
type ListBackupsResponse struct {
	BaseResponse
	Data struct {
		Backups []*Backup `json:"backups"`
	} `json:"data"`
}

// IngressResponse
type IngressResponse struct {
	BaseResponse
	Data struct {
		IngressEntry string `json:"ingress_entry"`
	} `json:"data"`
}

// Client is a client for the Hassio API
type Client struct {
	client *http.Client
	token  string
	url    string
}

// NewService initializes and returns a new Hassio Client
func NewService(token string) *Client {
	return &Client{
		token: token,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

// handleResponse is a helper function to handle the response and error checking
func handleResponse(resp *http.Response, data interface{}) error {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, respBody)
	}

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	var baseResponse BaseResponse
	if err := json.NewDecoder(bytes.NewReader(respBody)).Decode(&baseResponse); err != nil {
		return fmt.Errorf("could not parse response: %v", err)
	}

	// Check if the result is "ok"
	if baseResponse.Result != "ok" {
		return fmt.Errorf("error from API: %s", baseResponse.Message)
	}

	// If result is not nil, decode the specific response
	if data != nil {
		if err := json.NewDecoder(bytes.NewReader(respBody)).Decode(data); err != nil {
			return fmt.Errorf("could not parse data: %v", err)
		}
	}

	return nil
}

// GetBackup retrieves the details of a specific backup by its slug
func (c *Client) GetBackup(slug string) (*Backup, error) {
	// Create the HTTP request
	url := fmt.Sprintf("http://supervisor/backups/%s/info", slug)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Perform the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var backupResponse BackupResponse
	if err := handleResponse(resp, &backupResponse); err != nil {
		return nil, err
	}

	rs := backupResponse.Data.Slug
	if rs != slug {
		return nil, errors.New("missing or invalid backup in response")
	}

	return &backupResponse.Data, nil
}

// ListBackups retrieves a list of all backups from Home Assistant
func (c *Client) ListBackups() ([]*Backup, error) {
	// Create the HTTP request
	url := "http://supervisor/backups"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Perform the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var backupResponse ListBackupsResponse
	if err := handleResponse(resp, &backupResponse); err != nil {
		return nil, err
	}

	return backupResponse.Data.Backups, nil
}

// BackupFull requests a full backup from Home Assistant
func (c *Client) BackupFull(name string) (string, error) {
	// Create the JSON body for the request
	jsonBody := []byte(fmt.Sprintf(`{"name": "%s"}`, name))
	bodyReader := bytes.NewReader(jsonBody)

	// Create the HTTP request
	url := "http://supervisor/backups/new/full"
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Perform the request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}

	// Read and parse the response
	var response BackupResponse
	if err := handleResponse(resp, &response); err != nil {
		return "", err
	}

	// Extract the slug from the response data
	slug := response.Data.Slug
	if slug == "" {
		return "", errors.New("missing or invalid slug in response")
	}

	return slug, nil
}

// UploadBackup uploads a backup file to Home Assistant
func (c *Client) UploadBackup(data io.Reader) error {
	// Create a buffer to hold the multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create the form file field
	part, err := writer.CreateFormFile("file", "temp")
	if err != nil {
		return err
	}

	// Copy the file content into the form field
	_, err = io.Copy(part, data)
	if err != nil {
		return err
	}

	// Close the multipart writer to finalize the form data
	err = writer.Close()
	if err != nil {
		return err
	}

	// Create the HTTP request
	url := "http://supervisor/backups/new/upload"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Perform the request
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// DeleteBackup requests a specific backup to be deleted from Home Assistant
func (c *Client) DeleteBackup(slug string) error {
	// Create the HTTP request
	url := fmt.Sprintf("http://supervisor/backups/%s", slug)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Perform the request
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// RestoreBackup requests a specific backup to be restored in Home Assistant
func (c *Client) RestoreBackup(slug string) error {
	// Create the HTTP request
	url := fmt.Sprintf("http://supervisor/backups/%s/restore/full", slug)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Perform the request
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	return handleResponse(resp, nil)
}

// GetIngressEntry returns the hassio ingress path for the addon
func GetIngressEntry(token string) (string, error) {
	bearer := "Bearer " + token

	// Create the HTTP request
	url := "http://supervisor/addons/self/info"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Add authorization header to the request
	req.Header.Add("Authorization", bearer)

	// Send request using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	// Read and parse the response
	var baseResponse BaseResponse
	if err := handleResponse(resp, &baseResponse); err != nil {
		return "", err
	}

	// Extract the ingress_entry from the response data
	ingressEntry, ok := baseResponse.Data.(map[string]interface{})["ingress_entry"].(string)
	if !ok {
		return "", errors.New("missing or invalid ingress_entry in response")
	}

	fmt.Println("Ingress Entry: ", ingressEntry)

	return ingressEntry, nil
}
