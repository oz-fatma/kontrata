package agent

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/oz-fatma/kontrata/backend/internal/extract"
	"github.com/oz-fatma/kontrata/backend/internal/llm"
	"github.com/oz-fatma/kontrata/backend/internal/mask"
)

const parseError = "model çıktısı JSON olarak çözülemedi"

// Reader PDF sayfa metinlerinden şemaya uygun JSON çıkarır.
type Reader struct {
	LLM          llm.Client
	DumpDir      string
	ContractID   string
	SystemPrompt string
	dumpN        int
}

func (r *Reader) systemPrompt() string {
	if r != nil && strings.TrimSpace(r.SystemPrompt) != "" {
		return r.SystemPrompt
	}
	return SYSTEM_PROMPT
}

// Extract sayfaları birleştirir, modeli çağırır, onarır, doğrular.
// Şema hatasında bir düzeltme turu daha çalışır.
func (r *Reader) Extract(ctx context.Context, pages []string) (*ExtractResult, error) {
	start := time.Now()
	out := &ExtractResult{Data: map[string]any{}}
	if r == nil || r.LLM == nil {
		out.SchemaErrors = []string{"model yapılandırılmadı"}
		out.Duration = time.Since(start)
		return out, nil
	}

	masked := mask.Apply(joinPages(pages))
	log.Printf("maskeleme uygulandi alan=%d", masked.Count)
	sys := r.systemPrompt()
	raw, err := r.LLM.Generate(ctx, sys, masked.Text)
	if err != nil {
		out.Duration = time.Since(start)
		return out, err
	}
	r.dumpExchange(masked.Text, raw)

	data, repairs, errs := decodeAndCheck(raw)
	if len(errs) > 0 {
		out.RetryCount = 1
		corr := correctionPrompt(masked.Text, errs)
		raw2, err := r.LLM.Generate(ctx, sys, corr)
		if err == nil {
			r.dumpExchange(corr, raw2)
			data, repairs, errs = decodeAndCheck(raw2)
		}
	}

	attachMeta(data, repairs, pages)
	out.Data = data
	out.Repairs = repairs
	out.SchemaErrors = errs
	out.Meta = BuildExtractionMeta(data, repairs, pages)
	out.Duration = time.Since(start)
	return out, nil
}

func (r *Reader) dumpExchange(sent, received string) {
	if r == nil || r.DumpDir == "" {
		return
	}
	r.dumpN++
	id := sanitizeDumpID(r.ContractID)
	name := "cikarma-" + id + "-" + time.Now().UTC().Format("20060102T150405Z") + "-" + strconv.Itoa(r.dumpN) + ".txt"
	path := filepath.Join(r.DumpDir, name)
	if err := os.WriteFile(path, []byte(formatLLMDump(sent, received)), 0o600); err != nil {
		log.Printf("llm dump yazılamadı: %v", err)
		return
	}
	log.Printf("llm dump yazildi gonderilen=%d alinan=%d", utf8.RuneCountInString(sent), utf8.RuneCountInString(received))
}

func formatLLMDump(sent, received string) string {
	var b strings.Builder
	b.WriteString("=== GONDERILEN (maskelenmis) ===\n")
	b.WriteString(sent)
	if !strings.HasSuffix(sent, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("\n=== ALINAN ===\n")
	b.WriteString(received)
	if received != "" && !strings.HasSuffix(received, "\n") {
		b.WriteByte('\n')
	}
	return b.String()
}

func sanitizeDumpID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "bilinmeyen"
	}
	var b strings.Builder
	for _, c := range id {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	if b.Len() == 0 {
		return "bilinmeyen"
	}
	return b.String()
}

func joinPages(pages []string) string {
	return strings.Join(pages, "\n\n")
}

func decodeAndCheck(raw string) (map[string]any, []string, []string) {
	parsed, err := extract.RepairJSON(raw)
	if err != nil {
		data, repairs := extract.Normalize(map[string]any{})
		return data, repairs, []string{parseError}
	}
	data, repairs := extract.Normalize(parsed)
	return data, repairs, extract.Validate(data)
}

func correctionPrompt(contract string, errs []string) string {
	var b strings.Builder
	b.WriteString(contract)
	b.WriteString("\n\nÖnceki çıktı şema hataları:\n")
	for _, e := range errs {
		b.WriteString("- ")
		b.WriteString(e)
		b.WriteByte('\n')
	}
	b.WriteString("Düzeltilmiş tek JSON nesnesi üret. Açıklama yazma.")
	return b.String()
}

func attachMeta(data map[string]any, repairs []string, pages []string) {
	if data == nil {
		return
	}
	meta := BuildExtractionMeta(data, repairs, pages)
	internal := make(map[string]any, len(meta))
	for _, m := range meta {
		entry := map[string]any{"guven": m.Confidence}
		if m.SourcePage != nil {
			entry["kaynak_sayfa"] = int(*m.SourcePage)
		}
		internal[m.SchemaPath] = entry
	}
	data["cikarim_meta"] = internal
}
