package config

import (
	"strings"
	"testing"
)

func TestLoad_DefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if cfg.Port != defaultPort {
		t.Fatalf("Port = %d, beklenen %d", cfg.Port, defaultPort)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	t.Setenv("MONGO_URI", "")

	_, err := Load()
	if err == nil {
		t.Fatal("eksik zorunlu değişken için hata bekleniyordu")
	}
	if !strings.Contains(err.Error(), "MONGO_URI") {
		t.Fatalf("hata mesajında değişken adı yok: %v", err)
	}
}
