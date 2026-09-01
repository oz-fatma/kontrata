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
	Port     int
	MongoURI string
}

// requiredNames açılışta dolu olması gereken ortam değişkenleri.
// Şimdilik boş; Aşama 2'de MONGO_URI eklenecek.
var requiredNames []string

// Load ortam değişkenlerini okur. Zorunlu bir değişken eksikse hata döner.
func Load() (Config, error) {
	port, err := parsePort(os.Getenv("PORT"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Port:     port,
		MongoURI: strings.TrimSpace(os.Getenv("MONGO_URI")),
	}

	if err := checkRequired(os.Getenv); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func checkRequired(getenv func(string) string) error {
	var missing []string
	for _, name := range requiredNames {
		if strings.TrimSpace(getenv(name)) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("zorunlu ortam değişkeni eksik: %s", strings.Join(missing, ", "))
	}
	return nil
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
