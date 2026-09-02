package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultPort = 8080

// Config ortam değişkenlerinden okunan ayarlardır.
type Config struct {
	Port       int
	MongoURI   string
	Playground bool
}

// Load ortam değişkenlerini okur. Zorunlu bir değişken eksikse hata döner.
func Load() (Config, error) {
	port, err := parsePort(os.Getenv("PORT"))
	if err != nil {
		return Config{}, err
	}

	mongoURI := strings.TrimSpace(os.Getenv("MONGO_URI"))
	if mongoURI == "" {
		return Config{}, fmt.Errorf("zorunlu ortam değişkeni eksik: MONGO_URI")
	}

	return Config{
		Port:       port,
		MongoURI:   mongoURI,
		Playground: parseBool(os.Getenv("GRAPHQL_PLAYGROUND")),
	}, nil
}

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parsePort(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultPort, nil
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("PORT geçersiz: %q (1-65535 arası bir sayı olmalı)", raw)
	}
	return port, nil
}

// String günlük için maskelenmiş özet döner. Hassas alanlar *** olur.
func (c Config) String() string {
	return fmt.Sprintf("PORT=%d MONGO_URI=%s", c.Port, maskSecret(c.MongoURI))
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	return "***"
}
