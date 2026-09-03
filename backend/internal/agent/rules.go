package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	SeverityCritical = "KRITIK"
	SeverityWarning  = "UYARI"
	SeverityInfo     = "BILGI"

	SourceRule  = "KURAL"
	SourceModel = "MODEL"

	CodeDateConflict           = "KURAL_TARIH_CELISKISI"
	CodeSubperiodOverflow      = "KURAL_ALT_DONEM_TASMA"
	CodePriceAllotmentMismatch = "KURAL_FIYAT_KONTENJAN_UYUSMAZLIGI"
	CodeMissingStopSale        = "KURAL_STOP_SALE_YOK"
	CodeReleaseUnreasonable    = "KURAL_RELEASE_MAKUL_DEGIL"
	CodeMissingRequired        = "KURAL_ZORUNLU_ALAN_EKSIK"
)

const dateLayout = "2006-01-02"

// runRules deterministik kural motorunu çalıştırır.
func runRules(data map[string]any) []Finding {
	if data == nil {
		data = map[string]any{}
	}
	var out []Finding
	out = append(out, ruleDateConflict(data)...)
	out = append(out, ruleSubperiodOverflow(data)...)
	out = append(out, rulePriceAllotmentMismatch(data)...)
	out = append(out, ruleMissingStopSale(data)...)
	out = append(out, ruleReleaseUnreasonable(data)...)
	out = append(out, ruleMissingRequired(data)...)
	return out
}

func ruleDateConflict(data map[string]any) []Finding {
	d := asMap(data["donem"])
	if d == nil {
		return nil
	}
	start, ok1 := parseDate(d["baslangic"])
	end, ok2 := parseDate(d["bitis"])
	if !ok1 || !ok2 {
		return nil
	}
	if !end.Before(start) {
		return nil
	}
	return []Finding{{
		Code:        CodeDateConflict,
		Title:       "Çelişkili sözleşme tarihi",
		Description: "Bitiş tarihi başlangıçtan önce. Dönem tarihlerini sözleşmeden doğrulayıp düzeltin.",
		Severity:    SeverityCritical,
		Source:      SourceRule,
		FieldPath:   "donem/bitis",
	}}
}

func ruleSubperiodOverflow(data map[string]any) []Finding {
	d := asMap(data["donem"])
	if d == nil {
		return nil
	}
	start, ok1 := parseDate(d["baslangic"])
	end, ok2 := parseDate(d["bitis"])
	if !ok1 || !ok2 {
		return nil
	}
	var out []Finding
	for _, item := range asSlice(d["alt_donemler"]) {
		sub := asMap(item)
		if sub == nil {
			continue
		}
		ss, okS := parseDate(sub["baslangic"])
		se, okE := parseDate(sub["bitis"])
		if !okS || !okE {
			continue
		}
		if ss.Before(start) || se.After(end) {
			name := strings.TrimSpace(asString(sub["ad"]))
			desc := "Bir alt dönem ana sözleşme döneminin dışında. Alt dönem tarihlerini ana döneme göre düzeltin."
			if name != "" {
				desc = "Alt dönem \"" + name + "\" ana sözleşme döneminin dışında. Tarihleri ana döneme göre düzeltin."
			}
			out = append(out, Finding{
				Code:        CodeSubperiodOverflow,
				Title:       "Alt dönem ana dönemin dışında",
				Description: desc,
				Severity:    SeverityWarning,
				Source:      SourceRule,
				FieldPath:   "donem/alt_donemler",
			})
		}
	}
	return out
}

func rulePriceAllotmentMismatch(data map[string]any) []Finding {
	allotted := map[string]struct{}{}
	for _, item := range asSlice(data["oda_kontenjanlari"]) {
		m := asMap(item)
		if m == nil {
			continue
		}
		tip := strings.TrimSpace(asString(m["oda_tipi"]))
		if tip == "" {
			continue
		}
		allotted[strings.ToLower(tip)] = struct{}{}
	}
	seen := map[string]struct{}{}
	var out []Finding
	for _, item := range asSlice(data["fiyatlar"]) {
		m := asMap(item)
		if m == nil {
			continue
		}
		tip := strings.TrimSpace(asString(m["oda_tipi"]))
		if tip == "" {
			continue
		}
		key := strings.ToLower(tip)
		if _, ok := allotted[key]; ok {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, Finding{
			Code:        CodePriceAllotmentMismatch,
			Title:       "Fiyat–kontenjan uyuşmazlığı",
			Description: "fiyatlar içindeki oda_tipi (" + tip + ") oda_kontenjanlari listesinde yok. Kontenjan tablosunu veya oda tipini düzeltin.",
			Severity:    SeverityCritical,
			Source:      SourceRule,
			FieldPath:   "fiyatlar/oda_tipi",
		})
	}
	return out
}

func ruleMissingStopSale(data map[string]any) []Finding {
	if len(asSlice(data["stop_sale"])) > 0 {
		return nil
	}
	return []Finding{{
		Code:        CodeMissingStopSale,
		Title:       "Stop-sale maddesi yok",
		Description: "stop_sale boş. Sözleşmede satış durdurma maddesi olup olmadığını kontrol edin.",
		Severity:    SeverityWarning,
		Source:      SourceRule,
		FieldPath:   "stop_sale",
	}}
}

func ruleReleaseUnreasonable(data map[string]any) []Finding {
	rel := asMap(data["release"])
	if rel == nil {
		return nil
	}
	gun, ok := asInt(rel["gun"])
	if !ok {
		return nil
	}
	if gun != 0 && gun <= 90 {
		return nil
	}
	return []Finding{{
		Code:        CodeReleaseUnreasonable,
		Title:       "Release süresi makul değil",
		Description: "release.gun 0 veya 90 günden büyük. Makul bir release süresi girin.",
		Severity:    SeverityWarning,
		Source:      SourceRule,
		FieldPath:   "release/gun",
	}}
}

func ruleMissingRequired(data map[string]any) []Finding {
	var out []Finding
	if isEmptyPeriod(data) || confidenceFilledDefault(data, "donem") {
		out = append(out, requiredFinding("donem"))
	}
	if isEmptySlice(data["oda_kontenjanlari"]) || confidenceFilledDefault(data, "oda_kontenjanlari") {
		out = append(out, requiredFinding("oda_kontenjanlari"))
	}
	if isEmptySlice(data["fiyatlar"]) || confidenceFilledDefault(data, "fiyatlar") {
		out = append(out, requiredFinding("fiyatlar"))
	}
	if isEmptyRelease(data) || confidenceFilledDefault(data, "release") {
		out = append(out, requiredFinding("release"))
	}
	// stop_sale boşluğu R4'te; burada yalnızca varsayılan doldurma (güven 0.2)
	if confidenceFilledDefault(data, "stop_sale") {
		out = append(out, requiredFinding("stop_sale"))
	}
	return out
}

func requiredFinding(path string) Finding {
	return Finding{
		Code:        CodeMissingRequired,
		Title:       "Zorunlu alan eksik",
		Description: path + " boş veya varsayılan değerle doldurulmuş. Sözleşmeden doğrulayın.",
		Severity:    SeverityCritical,
		Source:      SourceRule,
		FieldPath:   path,
	}
}

func isEmptyPeriod(data map[string]any) bool {
	d := asMap(data["donem"])
	if d == nil {
		return true
	}
	return isEmptyValue(d["baslangic"]) && isEmptyValue(d["bitis"])
}

func isEmptyRelease(data map[string]any) bool {
	rel := asMap(data["release"])
	if rel == nil {
		return true
	}
	_, ok := asInt(rel["gun"])
	return !ok
}

func isEmptySlice(v any) bool {
	return len(asSlice(v)) == 0
}

func confidenceFilledDefault(data map[string]any, schemaPath string) bool {
	meta := asMap(data["cikarim_meta"])
	if meta == nil {
		return false
	}
	entry := asMap(meta[schemaPath])
	if entry == nil {
		return false
	}
	g, ok := asFloat(entry["guven"])
	if !ok {
		return false
	}
	return g <= confidenceFilled+0.001
}

func parseDate(v any) (time.Time, bool) {
	s := strings.TrimSpace(asString(v))
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func isEmptyValue(v any) bool {
	if v == nil {
		return true
	}
	return strings.TrimSpace(asString(v)) == ""
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	switch s := v.(type) {
	case []any:
		return s
	default:
		return nil
	}
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprint(x)
	}
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			f, err2 := n.Float64()
			if err2 != nil {
				return 0, false
			}
			return int(f), true
		}
		return int(i), true
	default:
		return 0, false
	}
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
