// Package config membaca konfigurasi dari environment / file .env.
package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config = semua setting aplikasi.
type Config struct {
	Port string

	// DatabaseURL = connection string Neon POOLED (host -pooler) untuk server.
	DatabaseURL string
	// MigrationURL = connection string DIRECT untuk migrasi (fallback DatabaseURL).
	MigrationURL string
	// AutoMigrate = jalankan migrasi saat server start.
	AutoMigrate bool

	// APIKey = nilai header X-API-Key yang diterima.
	APIKey string
	// OwnerID = pemilik data untuk API key ini (mode single-owner).
	OwnerID string

	Location    *time.Location
	CORSOrigins []string

	LogLevel  string
	LogFormat string
}

// Load membaca .env (kalau ada) lalu environment variable.
func Load() (Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("info: file .env tidak ditemukan, memakai environment variable sistem")
	}

	cfg := Config{
		Port:         getEnv("PORT", "5080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		MigrationURL: getEnv("DATABASE_URL_UNPOOLED", os.Getenv("DATABASE_URL")),
		APIKey:       os.Getenv("API_KEY"),
		OwnerID:      getEnv("OWNER_ID", "owner"),
		CORSOrigins:  splitList(getEnv("CORS_ALLOWED_ORIGINS", "*")),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		LogFormat:    getEnv("LOG_FORMAT", "text"),
	}

	var err error
	if cfg.AutoMigrate, err = getBool("AUTO_MIGRATE", true); err != nil {
		return cfg, err
	}
	tz := getEnv("TIMEZONE", "Asia/Jakarta")
	if cfg.Location, err = time.LoadLocation(tz); err != nil {
		return cfg, fmt.Errorf("TIMEZONE %q tidak dikenal: %w", tz, err)
	}

	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL wajib diisi (connection string Neon)")
	}
	if cfg.APIKey == "" {
		return cfg, errors.New("API_KEY wajib diisi (samakan dengan VITE_API_KEY di josh-vault)")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getBool(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s harus true/false: %w", key, err)
	}
	return b, nil
}

func splitList(v string) []string {
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
