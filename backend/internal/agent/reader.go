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
)

const parseError = "model çıktısı JSON olarak çözülemedi"

// Reader PDF sayfa metinlerinden şemaya uygun JSON çıkarır.
type Reader struct {
	LLM        llm.Client
	DumpDir    string
	ContractID string
	dumpN      int
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

	user := joinPages(pages)
	raw, err := r.LLM.Generate(ctx, SYSTEM_PROMPT, user)
	if err != nil {
		out.Duration = time.Since(start)
		return out, err
	}
	r.dumpRaw(raw)

	data, repairs, errs := decodeAndCheck(raw)
	if len(errs) > 0 {
		out.RetryCount = 1
		corr := correctionPrompt(user, errs)
		raw2, err := r.LLM.Generate(ctx, SYSTEM_PROMPT, corr)
		if err == nil {
			r.dumpRaw(raw2)
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

func (r *Reader) dumpRaw(raw string) {
	if r == nil || r.DumpDir == "" {
		return
	}
	r.dumpN++
	id := sanitizeDumpID(r.ContractID)
	name := "cikarma-" + id + "-" + time.Now().UTC().Format("20060102T150405Z") + "-" + strconv.Itoa(r.dumpN) + ".txt"
	path := filepath.Join(r.DumpDir, name)
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		log.Printf("llm dump yazılamadı: %v", err)
		return
	}
	log.Printf("llm dump yazildi karakter=%d", utf8.RuneCountInString(raw))
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
