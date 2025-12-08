package torrent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
)

type TransmissionClient struct {
	cfg      *config.Config
	baseURL  string
	username string
	password string
	client   *http.Client
	sessionID string
}

type TransmissionRequest struct {
	Method    string      `json:"method"`
	Arguments interface{} `json:"arguments,omitempty"`
	Tag       int         `json:"tag,omitempty"`
}

type TransmissionResponse struct {
	Arguments interface{} `json:"arguments"`
	Result    string      `json:"result"`
}

type TransmissionTorrent struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Status          int     `json:"status"` // 0: Stopped, 4: Downloading, 6: Seeding
	PercentDone     float64 `json:"percentDone"`
	TotalSize       int64   `json:"totalSize"`
	DownloadedEver  int64   `json:"downloadedEver"`
	UploadedEver    int64   `json:"uploadedEver"`
	RateDownload    int64   `json:"rateDownload"`
	RateUpload      int64   `json:"rateUpload"`
	PeersSendingToUs int    `json:"peersSendingToUs"`
	PeersGettingFromUs int  `json:"peersGettingFromUs"`
	UploadRatio     float64 `json:"uploadRatio"`
	Eta             int     `json:"eta"`
}

func NewTransmissionClient(cfg *config.Config) (*TransmissionClient, error) {
	baseURL := cfg.TorrentClientPath
	if baseURL == "" {
		baseURL = "http://transmission:9091/transmission/rpc"
	}

	client := &TransmissionClient{
		cfg:      cfg,
		baseURL:  baseURL,
		username: "transmission",
		password: "transmission",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Obtenir le session ID
	if err := client.getSessionID(); err != nil {
		return nil, fmt.Errorf("erreur lors de l'obtention du session ID: %w", err)
	}

	return client, nil
}

func (c *TransmissionClient) getSessionID() error {
	req, err := http.NewRequest("GET", c.baseURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.sessionID = resp.Header.Get("X-Transmission-Session-Id")
	return nil
}

func (c *TransmissionClient) makeRequest(ctx context.Context, method string, arguments interface{}) (*TransmissionResponse, error) {
	reqBody := TransmissionRequest{
		Method:    method,
		Arguments: arguments,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Transmission-Session-Id", c.sessionID)
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Si 409, réessayer avec le nouveau session ID
	if resp.StatusCode == 409 {
		c.sessionID = resp.Header.Get("X-Transmission-Session-Id")
		return c.makeRequest(ctx, method, arguments)
	}

	var response TransmissionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Result != "success" {
		return nil, fmt.Errorf("erreur Transmission: %s", response.Result)
	}

	return &response, nil
}

func (c *TransmissionClient) AddTorrent(ctx context.Context, torrent *models.Torrent) error {
	var filename string
	if torrent.MagnetURL != "" {
		filename = torrent.MagnetURL
	} else if torrent.TorrentURL != "" {
		// Télécharger le fichier .torrent d'abord
		// TODO: Implémenter le téléchargement du fichier
		return fmt.Errorf("téléchargement de fichier .torrent non encore implémenté")
	} else {
		return fmt.Errorf("magnet URL ou torrent URL requis")
	}

	args := map[string]interface{}{
		"filename": filename,
		"download-dir": c.cfg.TorrentDownloadPath,
		"paused": false,
	}

	_, err := c.makeRequest(ctx, "torrent-add", args)
	return err
}

func (c *TransmissionClient) RemoveTorrent(ctx context.Context, torrentID string) error {
	id, err := strconv.Atoi(torrentID)
	if err != nil {
		return fmt.Errorf("ID de torrent invalide: %w", err)
	}

	args := map[string]interface{}{
		"ids":           []int{id},
		"delete-local-data": false,
	}

	_, err = c.makeRequest(ctx, "torrent-remove", args)
	return err
}

func (c *TransmissionClient) GetTorrentStatus(ctx context.Context, torrentID string) (*TorrentStatus, error) {
	id, err := strconv.Atoi(torrentID)
	if err != nil {
		return nil, fmt.Errorf("ID de torrent invalide: %w", err)
	}

	args := map[string]interface{}{
		"ids": []int{id},
		"fields": []string{
			"id", "name", "status", "percentDone", "totalSize",
			"downloadedEver", "uploadedEver", "rateDownload", "rateUpload",
			"peersSendingToUs", "peersGettingFromUs", "uploadRatio", "eta",
		},
	}

	resp, err := c.makeRequest(ctx, "torrent-get", args)
	if err != nil {
		return nil, err
	}

	// Parser la réponse
	argsMap, ok := resp.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("format de réponse invalide")
	}

	torrents, ok := argsMap["torrents"].([]interface{})
	if !ok || len(torrents) == 0 {
		return nil, fmt.Errorf("torrent non trouvé")
	}

	torrentData, ok := torrents[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("format de torrent invalide")
	}

	status := c.parseTorrentStatus(torrentData)
	return status, nil
}

func (c *TransmissionClient) ListTorrents(ctx context.Context) ([]*TorrentStatus, error) {
	args := map[string]interface{}{
		"fields": []string{
			"id", "name", "status", "percentDone", "totalSize",
			"downloadedEver", "uploadedEver", "rateDownload", "rateUpload",
			"peersSendingToUs", "peersGettingFromUs", "uploadRatio", "eta",
		},
	}

	resp, err := c.makeRequest(ctx, "torrent-get", args)
	if err != nil {
		return nil, err
	}

	argsMap, ok := resp.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("format de réponse invalide")
	}

	torrents, ok := argsMap["torrents"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("format de réponse invalide")
	}

	var statuses []*TorrentStatus
	for _, t := range torrents {
		torrentData, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		status := c.parseTorrentStatus(torrentData)
		statuses = append(statuses, status)
	}

	return statuses, nil
}

func (c *TransmissionClient) parseTorrentStatus(data map[string]interface{}) *TorrentStatus {
	status := &TorrentStatus{}

	if id, ok := data["id"].(float64); ok {
		status.ID = fmt.Sprintf("%.0f", id)
	}
	if name, ok := data["name"].(string); ok {
		status.Name = name
	}
	if percentDone, ok := data["percentDone"].(float64); ok {
		status.Progress = percentDone * 100
	}
	if totalSize, ok := data["totalSize"].(float64); ok {
		status.Size = int64(totalSize)
	}
	if downloaded, ok := data["downloadedEver"].(float64); ok {
		status.Downloaded = int64(downloaded)
	}
	if uploaded, ok := data["uploadedEver"].(float64); ok {
		status.Uploaded = int64(uploaded)
	}
	if rateDownload, ok := data["rateDownload"].(float64); ok {
		status.DownloadSpeed = int64(rateDownload)
	}
	if rateUpload, ok := data["rateUpload"].(float64); ok {
		status.UploadSpeed = int64(rateUpload)
	}
	if seeders, ok := data["peersSendingToUs"].(float64); ok {
		status.Seeders = int(seeders)
	}
	if leechers, ok := data["peersGettingFromUs"].(float64); ok {
		status.Leechers = int(leechers)
	}
	if ratio, ok := data["uploadRatio"].(float64); ok {
		status.Ratio = ratio
	}
	if eta, ok := data["eta"].(float64); ok {
		status.ETA = int64(eta)
	}

	// Convertir le statut Transmission en statut de notre modèle
	if statusInt, ok := data["status"].(float64); ok {
		switch int(statusInt) {
		case 0: // Stopped
			status.Status = models.TorrentStatusPaused
		case 4: // Downloading
			status.Status = models.TorrentStatusDownloading
		case 6: // Seeding
			status.Status = models.TorrentStatusSeeding
		default:
			status.Status = models.TorrentStatusPending
		}
	}

	return status
}

func (c *TransmissionClient) PauseTorrent(ctx context.Context, torrentID string) error {
	id, err := strconv.Atoi(torrentID)
	if err != nil {
		return fmt.Errorf("ID de torrent invalide: %w", err)
	}

	args := map[string]interface{}{
		"ids": []int{id},
	}

	_, err = c.makeRequest(ctx, "torrent-stop", args)
	return err
}

func (c *TransmissionClient) ResumeTorrent(ctx context.Context, torrentID string) error {
	id, err := strconv.Atoi(torrentID)
	if err != nil {
		return fmt.Errorf("ID de torrent invalide: %w", err)
	}

	args := map[string]interface{}{
		"ids": []int{id},
	}

	_, err = c.makeRequest(ctx, "torrent-start", args)
	return err
}

