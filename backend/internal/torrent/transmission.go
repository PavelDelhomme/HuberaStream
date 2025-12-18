package torrent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"streammake/internal/config"
	"streammake/internal/models"
)

type TransmissionClient struct {
	cfg       *config.Config
	baseURL   string
	username  string
	password  string
	client    *http.Client
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
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	Status             int     `json:"status"` // 0: Stopped, 4: Downloading, 6: Seeding
	PercentDone        float64 `json:"percentDone"`
	TotalSize          int64   `json:"totalSize"`
	DownloadedEver     int64   `json:"downloadedEver"`
	UploadedEver       int64   `json:"uploadedEver"`
	RateDownload       int64   `json:"rateDownload"`
	RateUpload         int64   `json:"rateUpload"`
	PeersSendingToUs   int     `json:"peersSendingToUs"`
	PeersGettingFromUs int     `json:"peersGettingFromUs"`
	UploadRatio        float64 `json:"uploadRatio"`
	Eta                int     `json:"eta"`
}

func NewTransmissionClient(cfg *config.Config) (*TransmissionClient, error) {
	baseURL := cfg.TorrentClientTransmissionURL
	if baseURL == "" {
		baseURL = "http://transmission:9091/transmission/rpc"
	}

	username := cfg.TorrentClientTransmissionUser
	if username == "" {
		username = "transmission"
	}

	password := cfg.TorrentClientTransmissionPass
	if password == "" {
		password = "transmission"
	}

	client := &TransmissionClient{
		cfg:      cfg,
		baseURL:  baseURL,
		username: username,
		password: password,
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
		// Vérifier que le magnet URL est bien formé
		if !strings.HasPrefix(torrent.MagnetURL, "magnet:") {
			return fmt.Errorf("magnet URL invalide: doit commencer par 'magnet:'")
		}
		// Vérifier qu'il contient un infohash (btih:)
		if !strings.Contains(strings.ToUpper(torrent.MagnetURL), "BTIH:") {
			return fmt.Errorf("magnet URL invalide: doit contenir un infohash (btih:)")
		}
		filename = torrent.MagnetURL
	} else if torrent.TorrentURL != "" {
		// Télécharger le fichier .torrent depuis l'URL
		// Transmission peut accepter une URL HTTP/HTTPS directement
		// Si c'est une URL de fichier .torrent, on peut l'utiliser directement
		if strings.HasSuffix(strings.ToLower(torrent.TorrentURL), ".torrent") {
			// URL directe vers un fichier .torrent
			filename = torrent.TorrentURL
		} else {
			// URL de page web, essayer de télécharger le fichier
			// Pour l'instant, on essaie l'URL directement (certains sites redirigent vers le fichier)
			filename = torrent.TorrentURL
		}
	} else {
		return fmt.Errorf("magnet URL ou torrent URL requis")
	}

	// Vérifier que le chemin de téléchargement est absolu
	// Le chemin doit être tel que Transmission le voit dans son conteneur
	downloadDir := c.cfg.TorrentDownloadPath
	if !strings.HasPrefix(downloadDir, "/") {
		return fmt.Errorf("download directory path is not absolute: %s", downloadDir)
	}
	
	// S'assurer que le chemin est bien /downloads pour Transmission
	// (c'est le chemin dans le conteneur Transmission, pas dans le backend)
	if downloadDir != "/downloads" && !strings.HasPrefix(downloadDir, "/downloads/") {
		log.Printf("[Transmission] ⚠️ Chemin de téléchargement: %s (attendu: /downloads)", downloadDir)
		// Utiliser /downloads par défaut si le chemin n'est pas correct
		downloadDir = "/downloads"
	}

	args := map[string]interface{}{
		"filename":     filename,
		"download-dir": downloadDir,
		"paused":       false,
	}

	filenamePreview := filename
	if len(filenamePreview) > 100 {
		filenamePreview = filenamePreview[:100] + "..."
	}
	log.Printf("[Transmission] Ajout du torrent: filename=%s, download-dir=%s", filenamePreview, downloadDir)
	response, err := c.makeRequest(ctx, "torrent-add", args)
	if err != nil {
		return fmt.Errorf("erreur lors de l'ajout au client: %w", err)
	}

	// Vérifier la réponse pour détecter les erreurs spécifiques
	if response.Arguments != nil {
		if argsMap, ok := response.Arguments.(map[string]interface{}); ok {
			if torrentDuplicate, ok := argsMap["torrent-duplicate"]; ok {
				return fmt.Errorf("torrent déjà présent dans le client: %v", torrentDuplicate)
			}
			if torrentAdded, ok := argsMap["torrent-added"]; ok {
				log.Printf("[Transmission] Torrent ajouté avec succès: %v", torrentAdded)
				// Le paramètre "paused": false devrait déjà démarrer le torrent
				// Attendre un peu pour que Transmission traite l'ajout et récupère les métadonnées
				if addedMap, ok := torrentAdded.(map[string]interface{}); ok {
					if id, ok := addedMap["id"].(float64); ok {
						torrentID := fmt.Sprintf("%.0f", id)
						log.Printf("[Transmission] Torrent ajouté avec ID: %s, attente de la récupération des métadonnées...", torrentID)
						// Pour un magnet link, Transmission doit d'abord récupérer les métadonnées
						// Cela peut prendre quelques secondes, donc on ne vérifie pas immédiatement
					}
				}
			}
		}
	}

	return nil
}

func (c *TransmissionClient) RemoveTorrent(ctx context.Context, torrentID string) error {
	id, err := strconv.Atoi(torrentID)
	if err != nil {
		return fmt.Errorf("ID de torrent invalide: %w", err)
	}

	args := map[string]interface{}{
		"ids":               []int{id},
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

	// Log la réponse brute pour debug
	log.Printf("[Transmission] Réponse brute pour torrent ID %s: %+v", torrentID, torrentData)

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
		// Ne logger que si la progression change significativement
		if status.Progress > 0 {
			log.Printf("[Transmission] parseTorrentStatus: Progress = %.2f%% (percentDone = %f)", status.Progress, percentDone)
		}
	}
	if totalSize, ok := data["totalSize"].(float64); ok {
		status.Size = int64(totalSize)
		// Si totalSize est 0, le torrent récupère encore les métadonnées (normal pour magnet)
		if status.Size > 0 {
			log.Printf("[Transmission] parseTorrentStatus: Size = %d bytes (%.2f MB)", status.Size, float64(status.Size)/(1024*1024))
		} else {
			log.Printf("[Transmission] parseTorrentStatus: Size = 0 (récupération des métadonnées en cours...)")
		}
	}
	if downloaded, ok := data["downloadedEver"].(float64); ok {
		status.Downloaded = int64(downloaded)
		// Ne logger que si quelque chose a été téléchargé
		if status.Downloaded > 0 {
			log.Printf("[Transmission] parseTorrentStatus: Downloaded = %d bytes (%.2f MB)", status.Downloaded, float64(status.Downloaded)/(1024*1024))
		}
	}
	if uploaded, ok := data["uploadedEver"].(float64); ok {
		status.Uploaded = int64(uploaded)
	}
	if rateDownload, ok := data["rateDownload"].(float64); ok {
		status.DownloadSpeed = int64(rateDownload)
		log.Printf("[Transmission] parseTorrentStatus: DownloadSpeed = %d bytes/s", status.DownloadSpeed)
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
		statusCode := int(statusInt)
		switch statusCode {
		case 0: // Stopped
			status.Status = models.TorrentStatusPaused
		case 4: // Downloading
			// Si totalSize est 0, le torrent récupère encore les métadonnées
			if status.Size == 0 {
				status.Status = models.TorrentStatusPending // Marquer comme pending pendant la récupération des métadonnées
				log.Printf("[Transmission] Torrent en récupération des métadonnées (status: %d, totalSize: 0)", statusCode)
			} else {
				status.Status = models.TorrentStatusDownloading
			}
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
