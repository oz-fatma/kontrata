package config

import (
	"strings"
	"testing"
	"time"
)

// Load'ın okuduğu tüm anahtarlar. Makefile/.env sızmasın diye her test
// bunları boşaltır; t.Setenv test bitince eski değeri geri yazar.
var loadEnvKeys = []string{
	"PORT", "MONGO_URI", "MONGO_DATABASE", "JWT_SECRET",
	"MAILER", "GRAPHQL_PLAYGROUND",
	"SMTP_PORT", "SMTP_HOST", "SMTP_USER", "SMTP_PASSWORD", "SMTP_FROM",
	"ARGON2_TIME", "ARGON2_MEMORY", "ARGON2_THREADS",
	"LLM_ENDPOINT_URL", "LLM_TOKEN", "LLM_MAX_TOKENS", "LLM_TIMEOUT_SECONDS", "LLM_DEBUG_DUMP", "UPLOAD_DIR",
}

func isolateEnv(t *testing.T) {
	t.Helper()
	for _, key := range loadEnvKeys {
		t.Setenv(key, "")
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	isolateEnv(t)
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-bytes!!")

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
	if cfg.UploadDir != "data/uploads" {
		t.Fatalf("UploadDir = %s", cfg.UploadDir)
	}
	if cfg.LLMEndpointURL != "" || cfg.LLMToken != "" {
		t.Fatalf("llm varsayılanı boş olmalı")
	}
	if cfg.LLMMaxTokens != 600 {
		t.Fatalf("LLMMaxTokens = %d", cfg.LLMMaxTokens)
	}
	if cfg.LLMTimeout != 240*time.Second {
		t.Fatalf("LLMTimeout = %s", cfg.LLMTimeout)
	}
	if cfg.LLMDebugDump {
		t.Fatal("LLMDebugDump varsayılan kapalı olmalı")
	}
}

func TestLoad_LLMDebugDump(t *testing.T) {
	isolateEnv(t)
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-bytes!!")
	t.Setenv("LLM_DEBUG_DUMP", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if !cfg.LLMDebugDump {
		t.Fatal("LLM_DEBUG_DUMP=true iken açık olmalı")
	}
	if !strings.Contains(cfg.String(), "LLM_DEBUG_DUMP=true") {
		t.Fatalf("özette bayrak yok: %s", cfg.String())
	}
}

func TestLoad_DoesNotLogSecrets(t *testing.T) {
	isolateEnv(t)
	t.Setenv("MONGO_URI", "mongodb://user:gizli@localhost:27017")
	t.Setenv("SMTP_PASSWORD", "smtp-gizli")
	t.Setenv("SMTP_FROM", "gizli@ornek.test")
	t.Setenv("JWT_SECRET", "super-secret-jwt-key-value")
	t.Setenv("LLM_TOKEN", "hf-gizli-token")
	t.Setenv("LLM_ENDPOINT_URL", "https://example.endpoints.huggingface.cloud")
	t.Setenv("UPLOAD_DIR", "data/uploads")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	s := cfg.String()
	for _, secret := range []string{"gizli", "smtp-gizli", "gizli@ornek.test", "super-secret-jwt-key-value", "hf-gizli-token"} {
		if strings.Contains(s, secret) {
			t.Fatalf("özet sır sızdırıyor")
		}
	}
	if !strings.Contains(s, "UPLOAD_DIR=data/uploads") {
		t.Fatalf("UPLOAD_DIR özet yok: %s", s)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	isolateEnv(t)
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-bytes!!")

	_, err := Load()
	if err == nil {
		t.Fatal("eksik zorunlu değişken için hata bekleniyordu")
	}
	if !strings.Contains(err.Error(), "MONGO_URI") {
		t.Fatalf("hata mesajında değişken adı yok: %v", err)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	isolateEnv(t)
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")

	_, err := Load()
	if err == nil {
		t.Fatal("eksik JWT_SECRET için hata bekleniyordu")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("hata mesajında değişken adı yok: %v", err)
	}
}
