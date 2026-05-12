package api

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// événements Radarr / Sonarr où un fichier disque est généralement présent
func arrEventImportsMedia(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "Download", "Import", "MovieFileImported", "EpisodeFileImported", "Upgrade":
		return true
	default:
		return false
	}
}

func extractMediaPathsFromArrWebhook(source string, body []byte) []string {
	var root map[string]interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		return nil
	}
	switch source {
	case "radarr":
		return pathsFromRadarr(root)
	case "sonarr":
		return pathsFromSonarr(root)
	default:
		return nil
	}
}

func pathsFromRadarr(m map[string]interface{}) []string {
	var out []string
	if mf, ok := m["MovieFile"].(map[string]interface{}); ok {
		if p, ok := mf["Path"].(string); ok && p != "" {
			out = append(out, p)
		}
	}
	return dedupePaths(out)
}

func pathsFromSonarr(m map[string]interface{}) []string {
	var out []string
	var seriesPath string
	if s, ok := m["Series"].(map[string]interface{}); ok {
		if p, ok := s["Path"].(string); ok {
			seriesPath = p
		}
	}
	if ef, ok := m["EpisodeFile"].(map[string]interface{}); ok {
		if p, ok := ef["Path"].(string); ok && p != "" {
			out = append(out, p)
		} else if rel, ok := ef["RelativePath"].(string); ok && rel != "" && seriesPath != "" {
			out = append(out, filepath.Join(seriesPath, rel))
		}
	}
	return dedupePaths(out)
}

func dedupePaths(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, p := range in {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// pickTitleFromArrWebhook extrait un titre lisible pour la bibliothèque StreamMake.
func pickTitleFromArrWebhook(source string, body []byte) string {
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return ""
	}
	switch source {
	case "radarr":
		if movie, ok := m["Movie"].(map[string]interface{}); ok {
			if t, ok := movie["Title"].(string); ok && t != "" {
				return t
			}
		}
	case "sonarr":
		if s, ok := m["Series"].(map[string]interface{}); ok {
			title, _ := s["Title"].(string)
			if ep, ok := m["Episodes"].([]interface{}); ok && len(ep) > 0 {
				if first, ok := ep[0].(map[string]interface{}); ok {
					if sn, ok := first["SeasonNumber"].(float64); ok {
						if en, ok := first["EpisodeNumber"].(float64); ok {
							return strings.TrimSpace(fmt.Sprintf("%s S%.0fE%.0f", title, sn, en))
						}
					}
				}
			}
			return title
		}
	}
	return ""
}
