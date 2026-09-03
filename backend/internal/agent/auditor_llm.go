package agent

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"unicode"

	"github.com/oz-fatma/kontrata/backend/internal/mask"
)

// AuditorMaxTokens Denetçi LLM çağrısının üst sınırıdır.
// Okuyucu LLM_MAX_TOKENS (600) kullanır; denetçi çıktısı kısa bir dizidir.
const AuditorMaxTokens = 300

const maxModelFindings = 5

const AUDITOR_SYSTEM_PROMPT = `Sen bir sözleşme denetçisisin. Sana bir kontenjan sözleşmesinin
metni ve ondan çıkarılmış JSON verilecek. Yalnızca YORUM
gerektiren sorunları bildir.

Şunları BİLDİRME (bunlar ayrı kontrol ediliyor):
- Tarih çelişkileri
- Fiyat ve kontenjan tablosu uyuşmazlığı
- Eksik stop-sale maddesi
- Eksik zorunlu alan

Şunları BİLDİR:
- Belirsiz ifade ("yaklaşık 10 gün", "approximately", "civarı",
  "gerektiğinde") — kesin değer verilmemiş
- Sözleşmede tanımsız bırakılmış koşul
- Birbiriyle çelişen madde metinleri
- Sektör normundan belirgin sapma

Çıktı SADECE JSON dizisi:
[{"baslik":"...","aciklama":"...","onem":"KRITIK|UYARI|BILGI","alan":"..."}]

Bulgu yoksa: []
En fazla 5 bulgu. Bittiğinde dur.`

type llmFindingRow struct {
	Baslik   string `json:"baslik"`
	Aciklama string `json:"aciklama"`
	Onem     string `json:"onem"`
	Alan     string `json:"alan"`
}

func (a *Auditor) llmFindings(ctx context.Context, data map[string]any, pages []string) []Finding {
	if a == nil || a.LLM == nil {
		return nil
	}
	user, ok := buildAuditorUser(data, pages)
	if !ok {
		return nil
	}
	masked := mask.Apply(user)
	log.Printf("maskeleme uygulandi alan=%d", masked.Count)
	raw, err := a.LLM.Generate(ctx, a.systemPrompt(), masked.Text)
	if err != nil {
		log.Printf("denetci llm atlandi")
		return nil
	}
	return parseLLMFindings(raw)
}

func buildAuditorUser(data map[string]any, pages []string) (string, bool) {
	payload := dataForLLM(data)
	encoded, err := json.Marshal(payload)
	if err != nil {
		log.Printf("denetci json kodlanamadi")
		return "", false
	}
	var b strings.Builder
	b.WriteString("Sözleşme metni:\n")
	b.WriteString(joinPages(pages))
	b.WriteString("\n\nÇıkarılmış JSON:\n")
	b.Write(encoded)
	return b.String(), true
}

func dataForLLM(data map[string]any) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(data))
	for k, v := range data {
		if k == "cikarim_meta" {
			continue
		}
		out[k] = v
	}
	return out
}

func parseLLMFindings(raw string) []Finding {
	span := jsonArraySpan(raw)
	if span == "" {
		return nil
	}
	var rows []llmFindingRow
	if err := json.Unmarshal([]byte(span), &rows); err != nil {
		return nil
	}
	out := make([]Finding, 0, len(rows))
	for i, row := range rows {
		if i >= maxModelFindings {
			break
		}
		title := strings.TrimSpace(row.Baslik)
		desc := strings.TrimSpace(row.Aciklama)
		if title == "" || desc == "" {
			continue
		}
		sev := normalizeSeverity(row.Onem)
		if sev == "" {
			continue
		}
		out = append(out, Finding{
			Code:        modelFindingCode(title, i),
			Title:       title,
			Description: desc,
			Severity:    sev,
			Source:      SourceModel,
			FieldPath:   strings.TrimSpace(row.Alan),
		})
	}
	return out
}

func normalizeSeverity(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case SeverityCritical:
		return SeverityCritical
	case SeverityWarning:
		return SeverityWarning
	case SeverityInfo:
		return SeverityInfo
	default:
		return ""
	}
}

func modelFindingCode(title string, i int) string {
	slug := slugCode(title)
	if slug == "" {
		return "MODEL_" + strconv.Itoa(i+1)
	}
	return "MODEL_" + slug
}

func slugCode(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	var b strings.Builder
	lastUS := false
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastUS = false
			n++
			if n >= 40 {
				break
			}
			continue
		}
		if !lastUS && b.Len() > 0 {
			b.WriteByte('_')
			lastUS = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func jsonArraySpan(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = stripFence(s)
	i := strings.Index(s, "[")
	j := strings.LastIndex(s, "]")
	if i < 0 || j < i {
		return ""
	}
	return s[i : j+1]
}

func stripFence(s string) string {
	trimmed := strings.TrimSpace(s)
	lower := strings.ToLower(trimmed)
	start := strings.Index(lower, "```")
	if start < 0 {
		return s
	}
	rest := trimmed[start+3:]
	if strings.HasPrefix(strings.ToLower(rest), "json") {
		rest = rest[4:]
	}
	rest = strings.TrimLeft(rest, " \t\r\n")
	end := strings.Index(rest, "```")
	if end < 0 {
		return rest
	}
	inner := strings.TrimSpace(rest[:end])
	if strings.Contains(inner, "[") {
		return inner
	}
	return s
}
