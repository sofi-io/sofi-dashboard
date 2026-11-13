package config

import (
	"fmt"
	"os"
)

type Config struct {
	Environment         string
	APIAddr             string
	DatabaseURL         string
	SOFIAPIBaseURL      string
	SOFIAPIKey          string
	ObjectStorageHost   string
	ObjectStorageBuck   string
	ObjectStorageUser   string
	ObjectStoragePass   string
	ObjectStorageSSL    bool
	DocumentStoragePath string
}

func Load() (Config, error) {
	cfg := Config{
		Environment:       getEnv("APP_ENV", "development"),
		APIAddr:           getEnv("API_ADDR", ":8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		SOFIAPIBaseURL:    os.Getenv("SOFI_API_BASE_URL"),
		SOFIAPIKey:        os.Getenv("SOFI_API_KEY"),
		ObjectStorageHost: os.Getenv("OBJECT_STORAGE_ENDPOINT"),
		ObjectStorageBuck: os.Getenv("OBJECT_STORAGE_BUCKET"),
		ObjectStorageUser: os.Getenv("OBJECT_STORAGE_ACCESS_KEY"),
		ObjectStoragePass: os.Getenv("OBJECT_STORAGE_SECRET_KEY"),
		ObjectStorageSSL:  getEnv("OBJECT_STORAGE_USE_SSL", "false") == "true",
		DocumentStoragePath: getEnv(
			"DOCUMENT_STORAGE_PATH",
			"./data/documents",
		),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
