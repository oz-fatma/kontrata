package agent

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/oz-fatma/kontrata/backend/internal/extract"
	"github.com/oz-fatma/kontrata/backend/internal/llm"
	"github.com/oz-fatma/kontrata/backend/internal/pdf"
)

func liveLLMClient(t *testing.T) llm.Client {
	t.Helper()
	for _, path := range []string{".env", "../../.env"} {
		if err := godotenv.Load(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s: %v", path, err)
		}
	}
	url := strings.TrimSpace(os.Getenv("LLM_ENDPOINT_URL"))
	if url == "" {
		t.Skip("LLM_ENDPOINT_URL yok")
	}
	token := strings.TrimSpace(os.Getenv("LLM_TOKEN"))
	maxTokens := 600
	if raw := strings.TrimSpace(os.Getenv("LLM_MAX_TOKENS")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			t.Fatalf("LLM_MAX_TOKENS geçersiz: %q", raw)
		}
		maxTokens = n
	}
	timeout := 240 * time.Second
	if raw := strings.TrimSpace(os.Getenv("LLM_TIMEOUT_SECONDS")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			t.Fatalf("LLM_TIMEOUT_SECONDS geçersiz: %q", raw)
		}
		timeout = time.Duration(n) * time.Second
	}
	primary := llm.NewHFEndpoint(url, token, maxTokens, timeout)
	url2 := strings.TrimSpace(os.Getenv("LLM_ENDPOINT_URL_2"))
	if url2 == "" {
		return primary
	}
	token2 := strings.TrimSpace(os.Getenv("LLM_TOKEN_2"))
	return llm.NewRouter([]llm.NamedClient{
		{Name: llm.EndpointUC1, Client: primary},
		{Name: llm.EndpointUC2, Client: llm.NewHFEndpoint(url2, token2, maxTokens, timeout)},
	}, llm.NopRecorder{})
}

func TestReader_LiveTUIAndCoral(t *testing.T) {
	if testing.Short() {
		t.Skip("kısa test modunda canlı LLM atlanır")
	}
	client := liveLLMClient(t)

	root := filepath.Clean(filepath.Join("..", "..", "..", "testdata", "sozlesmeler"))
	cases := []struct {
		name string
		file string
	}{
		{"tui", "tui-2026-yaz.pdf"},
		{"coral", "coral-bozuk.pdf"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(root, tc.file)
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("pdf açılamadı: %v", err)
			}
			defer func() { _ = f.Close() }()

			pages, err := pdf.ExtractText(f)
			if err != nil {
				t.Fatalf("pdf metni: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()

			start := time.Now()
			r := &Reader{LLM: client}
			res, err := r.Extract(ctx, pages)
			elapsed := time.Since(start)
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}

			status := "OK"
			if len(res.SchemaErrors) > 0 {
				status = "HATA"
			}
			encoded, _ := json.Marshal(res.Data)
			log.Printf("canli cikarma dosya=%s durum=%s sure=%s deneme=%d duzeltme=%d hata=%d json_uzunluk=%d",
				tc.file, status, elapsed, res.RetryCount+1, len(res.Repairs), len(res.SchemaErrors), len(encoded))
			if len(res.SchemaErrors) > 0 {
				log.Printf("canli cikarma hatalar dosya=%s: %v", tc.file, res.SchemaErrors)
			}
			if len(res.Repairs) > 0 {
				log.Printf("canli cikarma duzeltmeler dosya=%s: %v", tc.file, res.Repairs)
			}

			a := &Auditor{LLM: client}
			audit, err := a.Audit(ctx, res.Data, pages)
			if err != nil {
				t.Fatalf("Audit: %v", err)
			}
			ruleN, modelN := 0, 0
			for _, f := range audit.Findings {
				if f.Source == SourceRule {
					ruleN++
				} else {
					modelN++
				}
			}
			log.Printf("canli denetci dosya=%s kural=%d model=%d toplam=%d",
				tc.file, ruleN, modelN, len(audit.Findings))

			if status == "HATA" {
				t.Logf("şema hataları: %v", res.SchemaErrors)
			} else if errs := extract.Validate(res.Data); len(errs) > 0 {
				t.Fatalf("validate: %v", errs)
			}
		})
	}
}
