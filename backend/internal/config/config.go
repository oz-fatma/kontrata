package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/mailer"
)

const (
	defaultPort           = 8080
	defaultLLMMaxTokens   = 600
	defaultLLMTimeoutSecs = 240
)

// Config ortam değişkenlerinden okunan ayarlardır.
type Config struct {
	Port           int
	MongoURI       string
	MongoDatabase  string
	JWTSecret      []byte
	Playground     bool
	Mailer         string
	SMTP           mailer.SMTPConfig
	Argon2         auth.Params
	LLMEndpointURL string
	LLMToken       string
	LLMMaxTokens   int
	LLMTimeout     time.Duration
	LLMDebugDump   bool
	UploadDir      string
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

	mongoDatabase := strings.TrimSpace(os.Getenv("MONGO_DATABASE"))
	if mongoDatabase == "" {
		mongoDatabase = "kontrata"
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		return Config{}, fmt.Errorf("zorunlu ortam değişkeni eksik: JWT_SECRET")
	}

	mailerKind := strings.ToLower(strings.TrimSpace(os.Getenv("MAILER")))
	if mailerKind == "" {
		mailerKind = "console"
	}

	smtpPort := 587
	if raw := strings.TrimSpace(os.Getenv("SMTP_PORT")); raw != "" {
		smtpPort, err = parsePort(raw)
		if err != nil {
			return Config{}, fmt.Errorf("SMTP_PORT geçersiz")
		}
	}

	smtp := mailer.SMTPConfig{
		Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		Port:     smtpPort,
		User:     strings.TrimSpace(os.Getenv("SMTP_USER")),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}
	if mailerKind == "smtp" {
		if smtp.Host == "" {
			return Config{}, fmt.Errorf("zorunlu ortam değişkeni eksik: SMTP_HOST")
		}
		if smtp.From == "" {
			return Config{}, fmt.Errorf("zorunlu ortam değişkeni eksik: SMTP_FROM")
		}
	}

	params := auth.DefaultParams()
	if params.Time, err = parseUint32(os.Getenv("ARGON2_TIME"), params.Time); err != nil {
		return Config{}, fmt.Errorf("ARGON2_TIME geçersiz")
	}
	if params.Memory, err = parseUint32(os.Getenv("ARGON2_MEMORY"), params.Memory); err != nil {
		return Config{}, fmt.Errorf("ARGON2_MEMORY geçersiz")
	}
	var threads uint32
	if threads, err = parseUint32(os.Getenv("ARGON2_THREADS"), uint32(params.Threads)); err != nil {
		return Config{}, fmt.Errorf("ARGON2_THREADS geçersiz")
	}
	if threads > 255 {
		return Config{}, fmt.Errorf("ARGON2_THREADS geçersiz")
	}
	params.Threads = uint8(threads)

	maxTokens, err := parseUint32(os.Getenv("LLM_MAX_TOKENS"), defaultLLMMaxTokens)
	if err != nil {
		return Config{}, fmt.Errorf("LLM_MAX_TOKENS geçersiz")
	}
	timeoutSecs, err := parseUint32(os.Getenv("LLM_TIMEOUT_SECONDS"), defaultLLMTimeoutSecs)
	if err != nil {
		return Config{}, fmt.Errorf("LLM_TIMEOUT_SECONDS geçersiz")
	}

	return Config{
		Port:           port,
		MongoURI:       mongoURI,
		MongoDatabase:  mongoDatabase,
		JWTSecret:      []byte(jwtSecret),
		Playground:     parseBool(os.Getenv("GRAPHQL_PLAYGROUND")),
		Mailer:         mailerKind,
		SMTP:           smtp,
		Argon2:         params,
		LLMEndpointURL: strings.TrimSpace(os.Getenv("LLM_ENDPOINT_URL")),
		LLMToken:       strings.TrimSpace(os.Getenv("LLM_TOKEN")),
		LLMMaxTokens:   int(maxTokens),
		LLMTimeout:     time.Duration(timeoutSecs) * time.Second,
		LLMDebugDump:   parseBool(os.Getenv("LLM_DEBUG_DUMP")),
		UploadDir:      uploadDir(os.Getenv("UPLOAD_DIR")),
	}, nil
}

func uploadDir(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "data/uploads"
	}
	return raw
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

func parseUint32(raw string, fallback uint32) (uint32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("geçersiz sayı")
	}
	return uint32(n), nil
}

// String günlük için maskelenmiş özet döner. Hassas alanlar ve e-posta yazılmaz.
func (c Config) String() string {
	return fmt.Sprintf("PORT=%d MONGO_URI=%s MONGO_DATABASE=%s MAILER=%s JWT_SECRET=%s LLM_ENDPOINT_URL=%s LLM_TOKEN=%s LLM_MAX_TOKENS=%d LLM_TIMEOUT_SECONDS=%d LLM_DEBUG_DUMP=%t UPLOAD_DIR=%s",
		c.Port, maskSecret(c.MongoURI), c.MongoDatabase, c.Mailer, maskSecret(string(c.JWTSecret)),
		c.LLMEndpointURL, maskSecret(c.LLMToken), c.LLMMaxTokens, int(c.LLMTimeout/time.Second), c.LLMDebugDump, c.UploadDir)
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	return "***"
}
