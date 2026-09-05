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

func assertArgosChunkAShape(t *testing.T, data map[string]any) {
	t.Helper()
	for k := range data {
		if k != "donem" && k != "oda_kontenjanlari" {
			t.Fatalf("izin verilmeyen ust alan: %s", k)
		}
	}
	donem, ok := data["donem"].(map[string]any)
	if !ok {
		t.Fatalf("donem = %T", data["donem"])
	}
	for k := range donem {
		if k != "baslangic" && k != "bitis" && k != "alt_donemler" {
			t.Fatalf("izin verilmeyen donem alani: %s", k)
		}
	}
	alt, ok := donem["alt_donemler"].([]any)
	if !ok {
		t.Fatalf("alt_donemler = %T", donem["alt_donemler"])
	}
	if len(alt) != 0 {
		t.Fatalf("alt_donemler bos dizi olmali, got=%v", alt)
	}
}

func assertArgosOdaKontenjanlari(t *testing.T, data map[string]any) {
	t.Helper()
	want := map[string]int{
		"standart": 170,
		"suit":     20,
		"balayi":   5,
		"engelli":  1,
	}
	oda, ok := data["oda_kontenjanlari"].([]any)
	if !ok {
		t.Fatalf("oda_kontenjanlari = %T", data["oda_kontenjanlari"])
	}
	if len(oda) != len(want) {
		t.Fatalf("oda_kontenjanlari adet=%d beklenen=%d oda=%v", len(oda), len(want), data["oda_kontenjanlari"])
	}
	got := map[string]int{}
	for i, item := range oda {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("oda_kontenjanlari[%d] tip=%T", i, item)
		}
		tip, _ := m["oda_tipi"].(string)
		if tip == "" {
			t.Fatalf("oda_kontenjanlari[%d] oda_tipi bos", i)
		}
		if _, dup := got[tip]; dup {
			t.Fatalf("tekrarlayan oda_tipi: %s", tip)
		}
		adet, ok := m["adet"].(float64)
		if !ok {
			if n, ok := m["adet"].(int); ok {
				adet = float64(n)
			} else {
				t.Fatalf("oda_kontenjanlari[%d].adet sayi degil: %v", i, m["adet"])
			}
		}
		got[tip] = int(adet)
	}
	for tip, n := range want {
		if got[tip] != n {
			t.Fatalf("oda_tipi=%s adet=%d beklenen=%d tum=%v", tip, got[tip], n, got)
		}
	}
	b, _ := json.MarshalIndent(data["oda_kontenjanlari"], "", "  ")
	t.Logf("oda_kontenjanlari:\n%s", b)
}

func TestReader_LiveArgosMEGEP(t *testing.T) {
	if testing.Short() {
		t.Skip("kısa test modunda canlı LLM atlanır")
	}
	t.Setenv("EXTRACT_MODE", ExtractModeChunked)
	client := liveLLMClient(t)

	path := filepath.Clean(filepath.Join("..", "..", "..", "testdata", "sozlesmeler", "argos-megep.pdf"))
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

	r := &Reader{LLM: client}
	res, err := r.Extract(ctx, pages)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	assertArgosOdaKontenjanlari(t, res.Data)
	assertArgosMeta(t, res.Data)
	if donem, ok := res.Data["donem"].(map[string]any); ok {
		if alt, ok := donem["alt_donemler"].([]any); ok && len(alt) != 0 {
			t.Fatalf("birlesik donem.alt_donemler bos olmali, got=%v", alt)
		}
	}
	if len(res.SchemaErrors) > 0 {
		t.Fatalf("şema hataları: %v", res.SchemaErrors)
	}
	rel, _ := res.Data["release"].(map[string]any)
	if rel["gun"] != 10 {
		t.Fatalf("release.gun = %v tip=%T", rel["gun"], rel["gun"])
	}
}

func assertArgosMeta(t *testing.T, data map[string]any) {
	t.Helper()
	meta, ok := data["meta"].(map[string]any)
	if !ok || len(meta) == 0 {
		t.Fatalf("meta bos: %v", data["meta"])
	}
	for k := range meta {
		allowed := false
		for _, a := range chunkAMetaKeys {
			if k == a {
				allowed = true
				break
			}
		}
		if !allowed {
			t.Fatalf("meta uydurma alan: %s=%v", k, meta[k])
		}
	}
	otel, _ := meta["otel_adi"].(string)
	if !strings.Contains(strings.ToLower(otel), "argos") {
		t.Fatalf("meta.otel_adi = %q", otel)
	}
	if acente, _ := meta["acente_adi"].(string); acente != "" && !strings.Contains(strings.ToLower(acente), "side") {
		t.Fatalf("meta.acente_adi = %q", acente)
	}
	if pb, ok := meta["para_birimi"]; ok && pb != "GBP" {
		t.Fatalf("meta.para_birimi = %v", pb)
	}
	b, _ := json.MarshalIndent(meta, "", "  ")
	t.Logf("meta:\n%s", b)
}

func TestReader_LiveTUIChunked(t *testing.T) {
	if testing.Short() {
		t.Skip("kısa test modunda canlı LLM atlanır")
	}
	t.Setenv("EXTRACT_MODE", ExtractModeChunked)
	client := liveLLMClient(t)

	path := filepath.Clean(filepath.Join("..", "..", "..", "testdata", "sozlesmeler", "tui-2026-yaz.pdf"))
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

	r := &Reader{LLM: client}
	res, err := r.Extract(ctx, pages)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(res.SchemaErrors) > 0 {
		t.Fatalf("şema hataları: %v", res.SchemaErrors)
	}
	meta, ok := res.Data["meta"].(map[string]any)
	if !ok || len(meta) == 0 {
		t.Fatalf("meta bos: %v", res.Data["meta"])
	}
	for k := range meta {
		allowed := false
		for _, a := range chunkAMetaKeys {
			if k == a {
				allowed = true
				break
			}
		}
		if !allowed {
			t.Fatalf("meta uydurma alan: %s=%v", k, meta[k])
		}
	}
	otel, _ := meta["otel_adi"].(string)
	if !strings.Contains(strings.ToLower(otel), "kemer") {
		t.Fatalf("meta.otel_adi = %q", otel)
	}
	if pb, _ := meta["para_birimi"].(string); pb != "EUR" {
		t.Fatalf("meta.para_birimi = %q (EUR beklenir)", pb)
	}
	if acente, _ := meta["acente_adi"].(string); !strings.Contains(strings.ToLower(acente), "tui") {
		t.Fatalf("meta.acente_adi = %q", acente)
	}
	if _, ok := meta["sozlesme_tipi"]; ok {
		t.Fatalf("TUI metninde sozlesme_tipi olmamali: %v", meta["sozlesme_tipi"])
	}
	if _, ok := meta["sezon"]; ok {
		t.Fatalf("TUI dönem adlarından sezon uydurulmamali: %v", meta["sezon"])
	}
	oda, _ := res.Data["oda_kontenjanlari"].([]any)
	if len(oda) < 4 {
		t.Fatalf("oda_kontenjanlari adet=%d", len(oda))
	}
	b, _ := json.MarshalIndent(meta, "", "  ")
	t.Logf("meta:\n%s", b)
}
