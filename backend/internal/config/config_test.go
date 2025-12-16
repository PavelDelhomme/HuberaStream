package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Sauvegarder les variables d'environnement actuelles
	originalDBHost := os.Getenv("DB_HOST")
	originalJWTSecret := os.Getenv("JWT_SECRET")

	// Nettoyer après le test
	defer func() {
		if originalDBHost != "" {
			os.Setenv("DB_HOST", originalDBHost)
		} else {
			os.Unsetenv("DB_HOST")
		}
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	// Test avec valeurs par défaut
	cfg := Load()
	if cfg.DBHost == "" {
		t.Error("DBHost ne devrait pas être vide")
	}
	if cfg.JWTSecret == "" {
		t.Error("JWTSecret ne devrait pas être vide")
	}
	if cfg.DBPort == "" {
		t.Error("DBPort ne devrait pas être vide")
	}

	// Test avec variables d'environnement
	os.Setenv("DB_HOST", "test-host")
	os.Setenv("JWT_SECRET", "test-secret")

	cfg2 := Load()
	if cfg2.DBHost != "test-host" {
		t.Errorf("DBHost devrait être 'test-host', a obtenu: %s", cfg2.DBHost)
	}
	if cfg2.JWTSecret != "test-secret" {
		t.Errorf("JWTSecret devrait être 'test-secret', a obtenu: %s", cfg2.JWTSecret)
	}
}

func TestConfig_StoragePaths(t *testing.T) {
	cfg := Load()

	if cfg.VideoStoragePath == "" {
		t.Error("VideoStoragePath ne devrait pas être vide")
	}
	if cfg.ThumbnailStoragePath == "" {
		t.Error("ThumbnailStoragePath ne devrait pas être vide")
	}
	if cfg.TorrentStoragePath == "" {
		t.Error("TorrentStoragePath ne devrait pas être vide")
	}
}

func TestConfig_AdminConfig(t *testing.T) {
	cfg := Load()

	if cfg.AdminEmail == "" {
		t.Error("AdminEmail ne devrait pas être vide")
	}
	if cfg.AdminPassword == "" {
		t.Error("AdminPassword ne devrait pas être vide")
	}
}
