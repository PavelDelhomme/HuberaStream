package torrent

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"crypto/sha512"
	"golang.org/x/crypto/pbkdf2"
)

// ConfigureQBittorrentPassword configure automatiquement le mot de passe qBittorrent
func ConfigureQBittorrentPassword(ctx context.Context, url, username, password string) error {
	// Attendre que qBittorrent soit prêt
	if err := waitForQBittorrent(ctx, url); err != nil {
		return fmt.Errorf("qBittorrent n'est pas prêt: %w", err)
	}

	// Essayer de modifier directement le fichier de configuration
	configPaths := []string{
		"/config/qBittorrent/qBittorrent.conf",
		"/app/config/qBittorrent/qBittorrent.conf",
		filepath.Join(os.Getenv("HOME"), ".config/qBittorrent/qBittorrent.conf"),
	}

	var configPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		// Si on est dans Docker, chercher dans le volume
		// Le volume est monté à /config
		configPath = "/config/qBittorrent/qBittorrent.conf"
	}

	// Lire le fichier de configuration
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		// Si le fichier n'existe pas, essayer de le créer
		if os.IsNotExist(err) {
			return createQBittorrentConfig(configPath, username, password)
		}
		return fmt.Errorf("erreur lors de la lecture du fichier de configuration: %w", err)
	}

	// Modifier le fichier
	modified := false
	lines := strings.Split(string(configContent), "\n")
	
	for i, line := range lines {
		if strings.HasPrefix(line, "WebUI\\Username=") {
			lines[i] = fmt.Sprintf("WebUI\\Username=%s", username)
			modified = true
		} else if strings.HasPrefix(line, "WebUI\\Password_PBKDF2=") {
			// Générer le hash PBKDF2
			hash := generatePBKDF2Hash(password)
			lines[i] = fmt.Sprintf("WebUI\\Password_PBKDF2=%s", hash)
			modified = true
		}
	}

	// Si pas de ligne Password_PBKDF2, l'ajouter
	if !strings.Contains(string(configContent), "Password_PBKDF2") {
		// Trouver la ligne Username et ajouter après
		for i, line := range lines {
			if strings.HasPrefix(line, "WebUI\\Username=") {
				hash := generatePBKDF2Hash(password)
				lines = append(lines[:i+1], append([]string{fmt.Sprintf("WebUI\\Password_PBKDF2=%s", hash)}, lines[i+1:]...)...)
				modified = true
				break
			}
		}
	}

	if modified {
		// Sauvegarder le backup
		backupPath := configPath + ".backup"
		if err := os.WriteFile(backupPath, configContent, 0644); err != nil {
			return fmt.Errorf("erreur lors de la sauvegarde: %w", err)
		}

		// Écrire le nouveau fichier
		newContent := strings.Join(lines, "\n")
		if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("erreur lors de l'écriture: %w", err)
		}

		return nil
	}

	return fmt.Errorf("aucune modification nécessaire ou impossible de modifier")
}

// generatePBKDF2Hash génère un hash PBKDF2 compatible avec qBittorrent
func generatePBKDF2Hash(password string) string {
	// qBittorrent utilise PBKDF2 avec SHA-512, 100000 iterations
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		// Fallback sur un salt fixe si erreur
		salt = []byte("qBittorrentSalt")
	}

	// PBKDF2 avec SHA-512
			hash := pbkdf2.Key([]byte(password), salt, 100000, 64, sha512.New)
	
	// Encoder en base64 (salt + hash)
	combined := append(salt, hash...)
	return base64.StdEncoding.EncodeToString(combined)
}

// waitForQBittorrent attend que qBittorrent soit prêt
func waitForQBittorrent(ctx context.Context, url string) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url+"/api/v2/app/version", nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("qBittorrent n'est pas accessible après %d tentatives", maxRetries)
}

// createQBittorrentConfig crée un fichier de configuration qBittorrent
func createQBittorrentConfig(configPath, username, password string) error {
	// Créer le répertoire si nécessaire
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("erreur lors de la création du répertoire: %w", err)
	}

	// Générer le hash
	hash := generatePBKDF2Hash(password)

	// Créer le contenu de base
	content := fmt.Sprintf(`[LegalNotice]
Accepted=true

[Preferences]
WebUI\\Username=%s
WebUI\\Password_PBKDF2=%s
WebUI\\Address=*
WebUI\\Port=8080
`, username, hash)

	// Écrire le fichier
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("erreur lors de l'écriture: %w", err)
	}

	return nil
}

// ConfigureQBittorrentFromDocker configure qBittorrent depuis un conteneur Docker
func ConfigureQBittorrentFromDocker(ctx context.Context, containerName, username, password string) error {
	// Modifier le fichier de configuration dans le conteneur Docker
	configPath := "/config/qBittorrent/qBittorrent.conf"
	
	// Générer le hash
	hash := generatePBKDF2Hash(password)

	// Créer un script temporaire pour modifier la config
	script := fmt.Sprintf(`
		if [ -f %s ]; then
			# Sauvegarder
			cp %s %s.backup
			# Modifier
			sed -i "s/^WebUI\\\\Username=.*/WebUI\\\\Username=%s/" %s
			if grep -q "WebUI\\\\Password_PBKDF2=" %s; then
				sed -i "s/^WebUI\\\\Password_PBKDF2=.*/WebUI\\\\Password_PBKDF2=%s/" %s
			else
				echo "WebUI\\\\Password_PBKDF2=%s" >> %s
			fi
		else
			mkdir -p $(dirname %s)
			echo "[Preferences]" > %s
			echo "WebUI\\\\Username=%s" >> %s
			echo "WebUI\\\\Password_PBKDF2=%s" >> %s
			echo "WebUI\\\\Address=*" >> %s
			echo "WebUI\\\\Port=8080" >> %s
		fi
	`, configPath, configPath, configPath, username, configPath, configPath, configPath, hash, configPath, hash, configPath, configPath, configPath, username, configPath, hash, configPath, configPath)

	// Exécuter dans le conteneur
	cmd := exec.CommandContext(ctx, "docker", "exec", containerName, "sh", "-c", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("erreur lors de la modification de la configuration: %w", err)
	}

	return nil
}

