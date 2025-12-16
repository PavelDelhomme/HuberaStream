package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type Downloader struct {
	storagePath string
	client      *http.Client
}

type DownloadTask struct {
	ID        uuid.UUID
	URL       string
	VideoID   uuid.UUID
	Status    string
	Progress  float64
	FilePath  string
	Error     error
	CreatedAt time.Time
}

func NewDownloader(storagePath string) *Downloader {
	return &Downloader{
		storagePath: storagePath,
		client: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

func (d *Downloader) Download(ctx context.Context, url string, videoID uuid.UUID) (*DownloadTask, error) {
	task := &DownloadTask{
		ID:        uuid.New(),
		URL:       url,
		VideoID:   videoID,
		Status:    "downloading",
		Progress:  0,
		CreatedAt: time.Now(),
	}

	// Créer le nom de fichier
	fileName := fmt.Sprintf("%s_%s.mp4", videoID.String(), time.Now().Format("20060102_150405"))
	filePath := filepath.Join(d.storagePath, fileName)
	task.FilePath = filePath

	// Démarrer le téléchargement en goroutine
	go d.downloadFile(ctx, task)

	return task, nil
}

func (d *Downloader) downloadFile(ctx context.Context, task *DownloadTask) {
	defer func() {
		if task.Error != nil {
			task.Status = "error"
		} else {
			task.Status = "completed"
			task.Progress = 100
		}
	}()

	// Créer la requête HTTP
	req, err := http.NewRequestWithContext(ctx, "GET", task.URL, nil)
	if err != nil {
		task.Error = fmt.Errorf("erreur lors de la création de la requête: %w", err)
		return
	}

	// Ajouter des headers pour simuler un navigateur
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "video/webm,video/ogg,video/*;q=0.9,*/*;q=0.8")

	// Exécuter la requête
	resp, err := d.client.Do(req)
	if err != nil {
		task.Error = fmt.Errorf("erreur lors du téléchargement: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		task.Error = fmt.Errorf("code de statut HTTP: %d", resp.StatusCode)
		return
	}

	// Créer le fichier de destination
	file, err := os.Create(task.FilePath)
	if err != nil {
		task.Error = fmt.Errorf("erreur lors de la création du fichier: %w", err)
		return
	}
	defer file.Close()

	// Copier les données avec suivi de progression
	totalBytes := resp.ContentLength
	var writtenBytes int64

	buffer := make([]byte, 32*1024) // 32KB buffer
	for {
		select {
		case <-ctx.Done():
			task.Error = ctx.Err()
			os.Remove(task.FilePath)
			return
		default:
			nr, er := resp.Body.Read(buffer)
			if nr > 0 {
				nw, ew := file.Write(buffer[0:nr])
				if nw < 0 || nr < nw {
					nw = 0
					if ew == nil {
						ew = fmt.Errorf("écriture invalide")
					}
				}
				writtenBytes += int64(nw)
				if ew != nil {
					task.Error = ew
					return
				}
				if nr != nw {
					task.Error = io.ErrShortWrite
					return
				}

				// Mettre à jour la progression
				if totalBytes > 0 {
					task.Progress = float64(writtenBytes) / float64(totalBytes) * 100
				}
			}
			if er != nil {
				if er != io.EOF {
					task.Error = er
				}
				return
			}
		}
	}
}

// GetTaskStatus retourne le statut d'une tâche de téléchargement
func (d *Downloader) GetTaskStatus(taskID uuid.UUID) (*DownloadTask, error) {
	// TODO: Implémenter un système de stockage des tâches (Redis ou DB)
	// Pour l'instant, c'est une implémentation basique
	return nil, fmt.Errorf("non implémenté")
}
