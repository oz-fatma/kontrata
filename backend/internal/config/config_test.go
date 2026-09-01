package config

import (
	"strings"
	"testing"
)

func TestLoad_DefaultPort(t *testing.T) {
	t.Setenv("PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if cfg.Port != defaultPort {
		t.Fatalf("Port = %d, beklenen %d", cfg.Port, defaultPort)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	previous := requiredNames
	requiredNames = []string{"MUST_BE_SET"}
	t.Cleanup(func() { requiredNames = previous })

	t.Setenv("MUST_BE_SET", "")
	_, err := Load()
	if err == nil {
		t.Fatal("eksik zorunlu değişken için hata bekleniyordu")
	}
	if !strings.Contains(err.Error(), "MUST_BE_SET") {
		t.Fatalf("hata mesajında değişken adı yok: %v", err)
	}
}
