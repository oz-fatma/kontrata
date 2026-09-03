package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oz-fatma/kontrata/backend/internal/llm"
)

const validJSON = `{
  "donem": {"baslangic": "2026-04-01", "bitis": "2026-10-31"},
  "oda_kontenjanlari": [{"oda_tipi": "standart", "adet": 10}],
  "fiyatlar": [{"oda_tipi": "standart", "tutar": 85, "birim": "oda_gecelik", "pansiyon": "HB"}],
  "release": {"gun": 21, "kapsam": "kontenjan_iadesi"},
  "stop_sale": []
}`

type stubLLM struct {
	responses []string
	err       error
	calls     int
	users     []string
}

func (s *stubLLM) Generate(_ context.Context, _, userPrompt string) (string, error) {
	s.users = append(s.users, userPrompt)
	if s.calls >= len(s.responses) {
		if s.err != nil {
			return "", s.err
		}
		return "", llm.ErrUnavailable
	}
	out := s.responses[s.calls]
	s.calls++
	return out, nil
}

func TestReader_CleanOutput(t *testing.T) {
	stub := &stubLLM{responses: []string{validJSON}}
	r := &Reader{LLM: stub}
	pages := []string{"Sezon 1 Nisan 2026 - 31 Ekim 2026. Standart oda 10 adet, 85 EUR."}
	res, err := r.Extract(context.Background(), pages)
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if res.RetryCount != 0 {
		t.Fatalf("RetryCount = %d", res.RetryCount)
	}
	if len(res.SchemaErrors) != 0 {
		t.Fatalf("şema hataları: %v", res.SchemaErrors)
	}
	if stub.calls != 1 {
		t.Fatalf("çağrı = %d", stub.calls)
	}
	donem, _ := res.Data["donem"].(map[string]any)
	if donem["baslangic"] != "2026-04-01" {
		t.Fatalf("donem.baslangic = %v", donem["baslangic"])
	}
}

func TestReader_RepairRound(t *testing.T) {
	broken := "açıklama\n```json\n{\"donem\": {\"baslangic\": \"2026-04-01\", \"bitis\": \"2026-10-31\"}}\n```"
	stub := &stubLLM{responses: []string{broken, validJSON}}
	r := &Reader{LLM: stub}
	res, err := r.Extract(context.Background(), []string{"standart oda kontenjanı 10"})
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if res.RetryCount != 1 {
		t.Fatalf("RetryCount = %d", res.RetryCount)
	}
	if len(res.SchemaErrors) != 0 {
		t.Fatalf("düzeltme sonrası hata: %v", res.SchemaErrors)
	}
	if stub.calls != 2 {
		t.Fatalf("çağrı = %d", stub.calls)
	}
	if !strings.Contains(stub.users[1], "Önceki çıktı şema hataları") {
		t.Fatal("düzeltme turunda hata listesi gönderilmedi")
	}
}

func TestReader_FailsTwice(t *testing.T) {
	stub := &stubLLM{responses: []string{"bu json değil", "yine değil"}}
	r := &Reader{LLM: stub}
	res, err := r.Extract(context.Background(), []string{"metin"})
	if err != nil {
		t.Fatalf("elde kalan kayıt için hata dönülmemeli: %v", err)
	}
	if res.RetryCount != 1 {
		t.Fatalf("RetryCount = %d", res.RetryCount)
	}
	if len(res.SchemaErrors) == 0 {
		t.Fatal("şema hataları bekleniyordu")
	}
	if res.Data == nil {
		t.Fatal("elde kalan veri boş")
	}
}

func TestReader_DumpsRawWhenConfigured(t *testing.T) {
	dir := t.TempDir()
	stub := &stubLLM{responses: []string{validJSON}}
	r := &Reader{LLM: stub, DumpDir: dir, ContractID: "abc123"}
	_, err := r.Extract(context.Background(), []string{"sayfa"})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) == 0 {
		t.Fatal("dump dosyası yok")
	}
	name := ents[0].Name()
	if !strings.HasPrefix(name, "cikarma-abc123-") || !strings.HasSuffix(name, "-1.txt") {
		t.Fatalf("dump adı beklenen değil: %s", name)
	}
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	if !strings.Contains(body, "=== GONDERILEN (maskelenmis) ===") {
		t.Fatalf("gönderilen bölüm yok: %s", body)
	}
	if !strings.Contains(body, "=== ALINAN ===") {
		t.Fatalf("alınan bölüm yok: %s", body)
	}
	if !strings.Contains(body, `"donem"`) {
		t.Fatalf("dump çıktısı beklenen değil: %s", body)
	}
	sent, recv, ok := strings.Cut(body, "=== ALINAN ===")
	if !ok {
		t.Fatal("bölümler ayrılamadı")
	}
	if !strings.Contains(sent, "sayfa") {
		t.Fatalf("gönderilen prompt yok: %s", sent)
	}
	if strings.Contains(sent, `"donem"`) {
		t.Fatal("model çıktısı gönderilen bölüme karıştı")
	}
	if !strings.Contains(recv, `"donem"`) {
		t.Fatalf("alınan çıktı yok: %s", recv)
	}
}

func TestReader_NoDumpByDefault(t *testing.T) {
	dir := t.TempDir()
	stub := &stubLLM{responses: []string{validJSON}}
	r := &Reader{LLM: stub}
	_, err := r.Extract(context.Background(), []string{"sayfa"})
	if err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 0 {
		t.Fatal("debug kapalıyken dump yazılmamalı")
	}
}

func TestReader_DumpUsesMaskedPrompt(t *testing.T) {
	dir := t.TempDir()
	stub := &stubLLM{responses: []string{validJSON}}
	r := &Reader{LLM: stub, DumpDir: dir, ContractID: "pii1"}
	_, err := r.Extract(context.Background(), []string{
		"İletişim ayse.demir@argosotel.com ve +90 242 813 45\n67 kimlik 12345678901",
	})
	if err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) == 0 {
		t.Fatal("dump dosyası yok")
	}
	b, err := os.ReadFile(filepath.Join(dir, ents[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	sent, _, ok := strings.Cut(body, "=== ALINAN ===")
	if !ok {
		t.Fatalf("bölüm yok: %s", body)
	}
	for _, leak := range []string{"ayse.demir@argosotel.com", "+90 242", "12345678901"} {
		if strings.Contains(sent, leak) {
			t.Fatalf("dump sızıntı %q: %s", leak, sent)
		}
	}
	if !strings.Contains(sent, "[EPOSTA]") || !strings.Contains(sent, "[TELEFON]") || !strings.Contains(sent, "[TCKN]") {
		t.Fatalf("maske jetonu yok: %s", sent)
	}
}
