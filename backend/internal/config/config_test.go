package config

import (
	"strings"
	"testing"
)

func TestLoad_DefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("MAILER", "")
	t.Setenv("MONGO_DATABASE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if cfg.Port != defaultPort {
		t.Fatalf("Port = %d, beklenen %d", cfg.Port, defaultPort)
	}
	if cfg.Mailer != "console" {
		t.Fatalf("Mailer = %s", cfg.Mailer)
	}
	if cfg.MongoDatabase != "kontrata" {
		t.Fatalf("MongoDatabase = %s", cfg.MongoDatabase)
	}
}

func TestLoad_DoesNotLogSecrets(t *testing.T) {
	t.Setenv("MONGO_URI", "mongodb://user:gizli@localhost:27017")
	t.Setenv("SMTP_PASSWORD", "smtp-gizli")
	t.Setenv("SMTP_FROM", "gizli@ornek.test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	s := cfg.String()
	for _, secret := range []string{"gizli", "smtp-gizli", "gizli@ornek.test"} {
		if strings.Contains(s, secret) {
			t.Fatalf("özet sır sızdırıyor")
		}
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
