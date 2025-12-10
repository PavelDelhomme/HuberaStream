package torrent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
)

type DelugeClient struct {
	cfg      *config.Config
	baseURL  string
	password string
	client   *http.Client
}

type DelugeRequest struct {
	ID     int           `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

type DelugeResponse struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

func NewDelugeClient(cfg *config.Config) (*DelugeClient, error) {
	baseURL := cfg.TorrentClientDelugeURL
	if baseURL == "" {
		baseURL = cfg.TorrentClientPath
	}
	if baseURL == "" {
		baseURL = "http://deluge:8112/json"
	}

	password := cfg.TorrentClientDelugePass
	if password == "" {
		password = "deluge"
	}

	client := &DelugeClient{
		cfg:      cfg,
		baseURL:  baseURL,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Se connecter
	if err := client.connect(context.Background()); err != nil {
		return nil, fmt.Errorf("erreur lors de la connexion: %w", err)
	}

	return client, nil
}

func (c *DelugeClient) connect(ctx context.Context) error {
	req := DelugeRequest{
		ID:     1,
		Method: "auth.login",
		Params: []interface{}{c.password},
	}

	_, err := c.makeRequest(ctx, req)
	return err
}

func (c *DelugeClient) makeRequest(ctx context.Context, req DelugeRequest) (*DelugeResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la sérialisation JSON: %w", err)
	}

	// S'assurer que l'URL se termine par /json
	baseURL := c.baseURL
	if !strings.HasSuffix(baseURL, "/json") {
		if strings.HasSuffix(baseURL, "/") {
			baseURL = baseURL + "json"
		} else {
			baseURL = baseURL + "/json"
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création de la requête: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("erreur réseau: %w", err)
	}
	defer resp.Body.Close()

	// Lire le corps pour vérifier si c'est du JSON ou du HTML
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la lecture de la réponse: %w", err)
	}

	// Vérifier si la réponse est du HTML (erreur de configuration)
	if len(body) > 0 && body[0] == '<' {
		return nil, fmt.Errorf("réponse HTML reçue au lieu de JSON. L'URL doit se terminer par /json. URL utilisée: %s", baseURL)
	}

	var delugeResp DelugeResponse
	if err := json.Unmarshal(body, &delugeResp); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing JSON: %w, réponse: %s", err, string(body))
	}

	if delugeResp.Error != nil {
		return nil, fmt.Errorf("erreur Deluge: %v", delugeResp.Error)
	}

	return &delugeResp, nil
}

func (c *DelugeClient) AddTorrent(ctx context.Context, torrent *models.Torrent) error {
	var torrentData string
	if torrent.MagnetURL != "" {
		torrentData = torrent.MagnetURL
	} else if torrent.TorrentURL != "" {
		// TODO: Télécharger le fichier .torrent
		return fmt.Errorf("téléchargement de fichier .torrent non encore implémenté")
	} else {
		return fmt.Errorf("magnet URL ou torrent URL requis")
	}

	req := DelugeRequest{
		ID:     2,
		Method: "core.add_torrent_magnet",
		Params: []interface{}{torrentData, map[string]interface{}{
			"download_location": c.cfg.TorrentDownloadPath,
		}},
	}

	_, err := c.makeRequest(ctx, req)
	return err
}

func (c *DelugeClient) RemoveTorrent(ctx context.Context, torrentID string) error {
	req := DelugeRequest{
		ID:     3,
		Method: "core.remove_torrent",
		Params: []interface{}{torrentID, false}, // false = ne pas supprimer les fichiers
	}

	_, err := c.makeRequest(ctx, req)
	return err
}

func (c *DelugeClient) GetTorrentStatus(ctx context.Context, torrentID string) (*TorrentStatus, error) {
	req := DelugeRequest{
		ID:     4,
		Method: "core.get_torrent_status",
		Params: []interface{}{torrentID, []string{
			"name", "state", "progress", "total_size", "total_done",
			"total_uploaded", "download_payload_rate", "upload_payload_rate",
			"num_seeds", "num_peers", "ratio", "eta",
		}},
	}

	resp, err := c.makeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	statusData, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("format de réponse invalide")
	}

	status := c.parseTorrentStatus(statusData)
	status.ID = torrentID
	return status, nil
}

func (c *DelugeClient) ListTorrents(ctx context.Context) ([]*TorrentStatus, error) {
	// Obtenir la liste des IDs
	req := DelugeRequest{
		ID:     5,
		Method: "core.get_session_state",
		Params: []interface{}{},
	}

	resp, err := c.makeRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	torrentIDs, ok := resp.Result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("format de réponse invalide")
	}

	var statuses []*TorrentStatus
	for _, id := range torrentIDs {
		torrentID, ok := id.(string)
		if !ok {
			continue
		}

		status, err := c.GetTorrentStatus(ctx, torrentID)
		if err != nil {
			continue
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

func (c *DelugeClient) parseTorrentStatus(data map[string]interface{}) *TorrentStatus {
	status := &TorrentStatus{}

	if name, ok := data["name"].(string); ok {
		status.Name = name
	}
	if progress, ok := data["progress"].(float64); ok {
		status.Progress = progress * 100
	}
	if totalSize, ok := data["total_size"].(float64); ok {
		status.Size = int64(totalSize)
	}
	if totalDone, ok := data["total_done"].(float64); ok {
		status.Downloaded = int64(totalDone)
	}
	if uploaded, ok := data["total_uploaded"].(float64); ok {
		status.Uploaded = int64(uploaded)
	}
	if dlSpeed, ok := data["download_payload_rate"].(float64); ok {
		status.DownloadSpeed = int64(dlSpeed)
	}
	if upSpeed, ok := data["upload_payload_rate"].(float64); ok {
		status.UploadSpeed = int64(upSpeed)
	}
	if seeds, ok := data["num_seeds"].(float64); ok {
		status.Seeders = int(seeds)
	}
	if peers, ok := data["num_peers"].(float64); ok {
		status.Leechers = int(peers)
	}
	if ratio, ok := data["ratio"].(float64); ok {
		status.Ratio = ratio
	}
	if eta, ok := data["eta"].(float64); ok {
		status.ETA = int64(eta)
	}

	// Convertir le statut Deluge
	if state, ok := data["state"].(string); ok {
		switch state {
		case "Downloading":
			status.Status = models.TorrentStatusDownloading
		case "Seeding":
			status.Status = models.TorrentStatusSeeding
		case "Paused":
			status.Status = models.TorrentStatusPaused
		default:
			status.Status = models.TorrentStatusPending
		}
	}

	return status
}

func (c *DelugeClient) PauseTorrent(ctx context.Context, torrentID string) error {
	req := DelugeRequest{
		ID:     6,
		Method: "core.pause_torrent",
		Params: []interface{}{[]string{torrentID}},
	}

	_, err := c.makeRequest(ctx, req)
	return err
}

func (c *DelugeClient) ResumeTorrent(ctx context.Context, torrentID string) error {
	req := DelugeRequest{
		ID:     7,
		Method: "core.resume_torrent",
		Params: []interface{}{[]string{torrentID}},
	}

	_, err := c.makeRequest(ctx, req)
	return err
}

