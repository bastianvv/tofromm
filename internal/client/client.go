package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
	"time"
)

// Client

type Client struct {
	BaseURL    string
	basicAuth  string
	HTTPClient *http.Client
}

func NewClient(baseURL, username, password string) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		basicAuth:  base64.StdEncoding.EncodeToString([]byte(username + ":" + password)),
		HTTPClient: &http.Client{Transport: transport},
	}
}

func (c *Client) authHeader() string {
	return "Basic " + c.basicAuth
}

// Platforms and ROMs

type Platform struct {
	ID       int    `json:"id"`
	Slug     string `json:"slug"`
	FsSlug   string `json:"fs_slug"`
	Name     string `json:"name"`
	RomCount int    `json:"rom_count"`
}

type Rom struct {
	ID             int    `json:"id"`
	FsName         string `json:"fs_name"`
	FsNameNoExt    string `json:"fs_name_no_ext"`
	Name           string `json:"name"`
	FsNameNoTags   string `json:"fs_name_no_tags"`
	PlatformID     int    `json:"platform_id"`
	PlatformFsSlug string `json:"platform_fs_slug"`
}

func (r *Rom) DisplayName() string {
	if r.Name != "" {
		return r.Name
	}
	if r.FsNameNoTags != "" {
		return r.FsNameNoTags
	}
	return r.FsNameNoExt
}

type RomPage struct {
	Items  []Rom `json:"items"`
	Total  int   `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// Device

type DeviceCreatePayload struct {
	Name          string  `json:"name"`
	Platform      *string `json:"platform"`
	Client        *string `json:"client"`
	ClientVersion *string `json:"client_version"`
	Hostname      *string `json:"hostname"`
	SyncMode      string  `json:"sync_mode"`
	AllowExisting bool    `json:"allow_existing"`
}

type DeviceCreateResponse struct {
	DeviceId  string  `json:"device_id"`
	Name      *string `json:"name"`
	CreatedAt string  `json:"created_at"`
}

// Saves

type Save struct {
	ID            int     `json:"id"`
	RomID         int     `json:"rom_id"`
	FileName      string  `json:"file_name"`
	FileNameNoExt string  `json:"file_name_no_ext"`
	FileExtension string  `json:"file_extension"`
	ContentHash   *string `json:"content_hash"`
	Emulator      *string `json:"emulator"`
	Slot          *string `json:"slot"`
	UpdatedAt     string  `json:"updated_at"`
}

type SlotSummary struct {
	Slot   *string `json:"slot"`
	Count  int     `json:"count"`
	Latest Save    `json:"latest"`
}

type SaveSummary struct {
	TotalCount int           `json:"total_count"`
	Slots      []SlotSummary `json:"slots"`
}

// Sync

type ClientSaveState struct {
	RomID         int     `json:"rom_id"`
	FileName      string  `json:"file_name"`
	Slot          *string `json:"slot,omitempty"`
	Emulator      *string `json:"emulator,omitempty"`
	ContentHash   *string `json:"content_hash,omitempty"`
	UpdatedAt     string  `json:"updated_at"`
	FileSizeBytes int64   `json:"file_size_bytes"`
}

type SyncNegotiatePayload struct {
	DeviceId string            `json:"device_id"`
	Saves    []ClientSaveState `json:"saves"`
}

type SyncOperation struct {
	Action            string  `json:"action"`
	RomID             int     `json:"rom_id"`
	SaveID            *int    `json:"save_id"`
	FileName          string  `json:"file_name"`
	Slot              *string `json:"slot"`
	Emulator          *string `json:"emulator"`
	Reason            string  `json:"reason"`
	ServerUpdatedAt   *string `json:"server_updated_at"`
	ServerContentHash *string `json:"server_content_hash"`
}

type SyncNegotiateResponse struct {
	SessionID     int             `json:"session_id"`
	Operations    []SyncOperation `json:"operations"`
	TotalUpload   int             `json:"total_upload"`
	TotalDownload int             `json:"total_download"`
	TotalConflict int             `json:"total_conflict"`
	TotalNoOp     int             `json:"total_no_op"`
}

type SyncCompleteRequest struct {
	OperationsCompleted int `json:"operations_completed"`
	OperationsFailed    int `json:"operations_failed"`
}

type SyncSession struct {
	ID                  int     `json:"id"`
	DeviceID            string  `json:"device_id"`
	Status              string  `json:"status"`
	InitiatedAt         string  `json:"initiated_at"`
	CompletedAt         *string `json:"completed_at"`
	OperationsPlanned   int     `json:"operations_planned"`
	OperationsCompleted int     `json:"operations_completed"`
	OperationsFailed    int     `json:"operations_failed"`
}

// HTTP methods

func (c *Client) GetPlatforms() ([]Platform, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/api/platforms", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Get Platforms: status %d, body: %s", resp.StatusCode, body)
	}

	var platforms []Platform
	return platforms, json.NewDecoder(resp.Body).Decode(&platforms)
}

func (c *Client) GetRomsByPlatform(platformID int) ([]Rom, error) {
	var all []Rom
	limit, offset := 500, 0

	for {
		url := fmt.Sprintf("%s/api/roms?platform_ids=%d&limit=%d&offset=%d", c.BaseURL, platformID, limit, offset)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", c.authHeader())

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("Get ROMs By Platform: status %d, body: %s", resp.StatusCode, body)
		}

		var page RomPage
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		all = append(all, page.Items...)
		offset += len(page.Items)
		if offset >= page.Total {
			break
		}
	}

	return all, nil
}

func (c *Client) RegisterDevice(hostname, platform, name string) (DeviceCreateResponse, error) {
	clientName := "tofromm"
	payload := DeviceCreatePayload{
		Name:          name,
		Hostname:      &hostname,
		Platform:      &platform,
		Client:        &clientName,
		SyncMode:      "api",
		AllowExisting: true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return DeviceCreateResponse{}, err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/devices", c.BaseURL), bytes.NewReader(body))
	if err != nil {
		return DeviceCreateResponse{}, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return DeviceCreateResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return DeviceCreateResponse{}, fmt.Errorf("Register Device: status %d, body: %s", resp.StatusCode, body)
	}

	var result DeviceCreateResponse
	return result, json.NewDecoder(resp.Body).Decode(&result)
}

func (c *Client) DownloadRom(romID int, fileName string, w io.Writer) error {
	url := fmt.Sprintf("%s/api/roms/%d/content/%s", c.BaseURL, romID, fileName)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Downloading ROM: status %d, body: %s", resp.StatusCode, body)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

func (c *Client) GetSavesSummary(romID int) (SaveSummary, error) {
	url := fmt.Sprintf("%s/api/saves/summary?rom_id=%d", c.BaseURL, romID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return SaveSummary{}, err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return SaveSummary{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return SaveSummary{}, fmt.Errorf("Get Saves Summary for ROM %d: status %d, body: %s", romID, resp.StatusCode, body)
	}

	var summary SaveSummary
	return summary, json.NewDecoder(resp.Body).Decode(&summary)
}

func (c *Client) DownloadSave(saveID int, w io.Writer) error {
	url := fmt.Sprintf("%s/api/saves/%d/content", c.BaseURL, saveID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Downloading Save %d: status %d, body: %s", saveID, resp.StatusCode, body)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

func (c *Client) ConfirmDownload(saveID int, deviceId string) error {
	payload := map[string]string{"device_id": deviceId}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/saves/%d/downloaded", c.BaseURL, saveID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Confirm Download for Save %d: status %d, body: %s", saveID, resp.StatusCode, body)
	}

	return nil
}

func (c *Client) UploadSave(romID int, deviceID string, fileName string, r io.Reader, overwrite bool) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	part, err := w.CreateFormFile("saveFile", fileName)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, r); err != nil {
		return err
	}
	w.Close()

	url := fmt.Sprintf("%s/api/saves?rom_id=%d&device_id=%s&overwrite=%t", c.BaseURL, romID, deviceID, overwrite)

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Upload Save for ROM %d: status %d, body: %s", romID, resp.StatusCode, body)
	}

	return nil
}

func (c *Client) Negotiate(deviceID string, saves []ClientSaveState) (SyncNegotiateResponse, error) {
	payload := SyncNegotiatePayload{DeviceId: deviceID, Saves: saves}

	body, err := json.Marshal(payload)
	if err != nil {
		return SyncNegotiateResponse{}, err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/sync/negotiate", bytes.NewReader(body))
	if err != nil {
		return SyncNegotiateResponse{}, err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return SyncNegotiateResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return SyncNegotiateResponse{}, fmt.Errorf("Negotiate Sync: status %d, body: %s", resp.StatusCode, body)
	}

	var result SyncNegotiateResponse
	return result, json.NewDecoder(resp.Body).Decode(&result)
}

func (c *Client) CompleteSession(sessionID, completed, failed int) error {
	payload := SyncCompleteRequest{OperationsCompleted: completed, OperationsFailed: failed}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/sync/sessions/%d/complete", c.BaseURL, sessionID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Complete Session: status %d, body: %s", resp.StatusCode, body)
	}

	return nil
}
