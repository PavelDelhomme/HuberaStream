package config

import (
	"os"
)

type Config struct {
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	RedisHost           string
	RedisPort           string
	JWTSecret           string
	VideoStoragePath    string
	ThumbnailStoragePath string
}

func Load() *Config {
	return &Config{
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "streammake"),
		DBPassword:          getEnv("DB_PASSWORD", "streammake_password"),
		DBName:              getEnv("DB_NAME", "streammake_db"),
		RedisHost:           getEnv("REDIS_HOST", "localhost"),
		RedisPort:           getEnv("REDIS_PORT", "6379"),
		JWTSecret:           getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		VideoStoragePath:    getEnv("VIDEO_STORAGE_PATH", "./storage/videos"),
		ThumbnailStoragePath: getEnv("THUMBNAIL_STORAGE_PATH", "./storage/thumbnails"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

