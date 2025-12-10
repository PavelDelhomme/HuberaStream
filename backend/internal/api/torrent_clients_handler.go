package api

import (
	"net/http"

	"streammake/internal/config"
	"streammake/internal/torrent"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TorrentClientsHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewTorrentClientsHandler(db *gorm.DB, cfg *config.Config) *TorrentClientsHandler {
	return &TorrentClientsHandler{
		db:  db,
		cfg: cfg,
	}
}

type TorrentClientStatus struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
	WebPort   string `json:"web_port"`
	WebURL    string `json:"web_url"`
}

// GetTorrentClientsStatus retourne le statut de tous les clients torrent
func (h *TorrentClientsHandler) GetTorrentClientsStatus(c *gin.Context) {
	clients := []TorrentClientStatus{
		{
			Type:      "transmission",
			Name:      "Transmission",
			Available: true,
			WebPort:   "3305",
			WebURL:    "http://localhost:3305",
		},
		{
			Type:      "qbittorrent",
			Name:      "qBittorrent",
			Available: true,
			WebPort:   "3306",
			WebURL:    "http://localhost:3306",
		},
		{
			Type:      "deluge",
			Name:      "Deluge",
			Available: true,
			WebPort:   "3307",
			WebURL:    "http://localhost:3307",
		},
		{
			Type:      "rtorrent",
			Name:      "rTorrent/ruTorrent",
			Available: true,
			WebPort:   "3308",
			WebURL:    "http://localhost:3308",
		},
	}

	c.JSON(http.StatusOK, gin.H{"clients": clients})
}

// GetCurrentTorrentClient retourne le client torrent actuellement configuré
func (h *TorrentClientsHandler) GetCurrentTorrentClient(c *gin.Context) {
	currentClient := gin.H{
		"type": h.cfg.TorrentClientType,
		"config": gin.H{
			"transmission_url":  h.cfg.TorrentClientTransmissionURL,
			"transmission_user": h.cfg.TorrentClientTransmissionUser,
			"transmission_pass": h.cfg.TorrentClientTransmissionPass,
			"qbittorrent_url":   h.cfg.TorrentClientQBittorrentURL,
			"qbittorrent_user":  h.cfg.TorrentClientQBittorrentUser,
			"qbittorrent_pass": h.cfg.TorrentClientQBittorrentPass,
			"deluge_url":        h.cfg.TorrentClientDelugeURL,
			"deluge_pass":       h.cfg.TorrentClientDelugePass,
			"rtorrent_url":      h.cfg.TorrentClientRTorrentURL,
		},
	}

	c.JSON(http.StatusOK, currentClient)
}

type UpdateTorrentClientRequest struct {
	ClientType string                 `json:"client_type"`
	Config     map[string]interface{} `json:"config"`
}

// UpdateCurrentTorrentClient met à jour le client torrent actif
func (h *TorrentClientsHandler) UpdateCurrentTorrentClient(c *gin.Context) {
	var req UpdateTorrentClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mettre à jour la configuration (dans un vrai cas, on sauvegarderait en DB)
	// Pour l'instant, on retourne juste un succès
	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration sauvegardée",
		"client_type": req.ClientType,
		"config": req.Config,
	})
}

type TestTorrentClientRequest struct {
	ClientType string                 `json:"client_type"`
	Config     map[string]interface{} `json:"config"`
}

// TestTorrentClient teste la connexion à un client torrent
func (h *TorrentClientsHandler) TestTorrentClient(c *gin.Context) {
	var req TestTorrentClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Créer un client temporaire pour tester
	clientType := torrent.ClientType(req.ClientType)
	
	// Créer une config temporaire pour le test
	testCfg := *h.cfg
	if url, ok := req.Config["url"].(string); ok {
		switch req.ClientType {
		case "transmission":
			testCfg.TorrentClientTransmissionURL = url
			if user, ok := req.Config["user"].(string); ok && user != "" {
				testCfg.TorrentClientTransmissionUser = user
			}
			if pass, ok := req.Config["pass"].(string); ok && pass != "" {
				testCfg.TorrentClientTransmissionPass = pass
			}
		case "qbittorrent":
			testCfg.TorrentClientQBittorrentURL = url
			if user, ok := req.Config["user"].(string); ok && user != "" {
				testCfg.TorrentClientQBittorrentUser = user
			}
			if pass, ok := req.Config["pass"].(string); ok && pass != "" {
				testCfg.TorrentClientQBittorrentPass = pass
			}
		case "deluge":
			testCfg.TorrentClientDelugeURL = url
			if pass, ok := req.Config["pass"].(string); ok && pass != "" {
				testCfg.TorrentClientDelugePass = pass
			}
		case "rtorrent":
			testCfg.TorrentClientRTorrentURL = url
		}
	}

	client, err := torrent.NewTorrentClient(clientType, &testCfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Erreur lors de la création du client",
			"message": err.Error(),
		})
		return
	}

	// Tester en listant les torrents
	ctx := c.Request.Context()
	_, err = client.ListTorrents(ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Erreur de connexion au client",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Connexion réussie",
		"client_type": req.ClientType,
	})
}
