package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"streammake/internal/config"
)

type MediaStackHandler struct {
	cfg *config.Config
}

func NewMediaStackHandler(cfg *config.Config) *MediaStackHandler {
	return &MediaStackHandler{cfg: cfg}
}

type arrProbe struct {
	URL        string `json:"url"`
	Configured bool   `json:"configured"`
	Reachable  bool   `json:"reachable"`
	Status     string `json:"status,omitempty"`
	Error      string `json:"error,omitempty"`
}

// GetMediaStackStatus vérifie la joignabilité HTTP des outils *arr (admin).
// Radarr/Sonarr/Prowlarr exposent leur propre API ; StreamMake ne les remplace pas :
// ce point d’entrée sert au monitoring et aux tests d’intégration.
func (h *MediaStackHandler) GetMediaStackStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 4*time.Second)
	defer cancel()

	out := gin.H{
		"prowlarr": probeArr(ctx, h.cfg.ProwlarrURL),
		"radarr":   probeArr(ctx, h.cfg.RadarrURL),
		"sonarr":   probeArr(ctx, h.cfg.SonarrURL),
		"note":     "Les *arr gèrent indexers + clients torrent ; l’API StreamMake reste la passerelle applicative (auth, vidéos, candidats).",
	}
	c.JSON(http.StatusOK, out)
}

func probeArr(ctx context.Context, raw string) arrProbe {
	p := arrProbe{URL: raw}
	if raw == "" {
		return p
	}
	p.Configured = true
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		p.Error = err.Error()
		return p
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		p.Error = err.Error()
		return p
	}
	defer resp.Body.Close()
	p.Reachable = true
	p.Status = resp.Status
	return p
}
