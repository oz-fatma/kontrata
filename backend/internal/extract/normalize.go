package extract

import (
	"encoding/json"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	currencySuffix = regexp.MustCompile(`(?i)(?:EUR|GBP|USD|TRY|TL|€|£|\$|₺)\s*$`)
	currencyPrefix = regexp.MustCompile(`(?i)^\s*(?:EUR|GBP|USD|TRY|TL|€|£|\$|₺)\s*`)
	euThousands    = regexp.MustCompile(`^(\d{1,3}(?:\.\d{3})+)(?:,(\d+))?$`)
	usThousands    = regexp.MustCompile(`^(\d{1,3}(?:,\d{3})+)(?:\.(\d+))?$`)
	euDecimal      = regexp.MustCompile(`^(\d+),(\d{1,2})$`)
	namedDate      = regexp.MustCompile(`(?i)^(\d{1,2})\s+([A-Za-zÇĞİÖŞÜçğıöşü]+)\s+(\d{4})$`)
)

var trMonths = map[string]time.Month{
	"ocak": time.January, "subat": time.February, "şubat": time.February,
	"mart": time.March, "nisan": time.April, "mayis": time.May, "mayıs": time.May,
	"haziran": time.June, "temmuz": time.July, "agustos": time.August, "ağustos": time.August,
	"eylul": time.September, "eylül": time.September, "ekim": time.October,
	"kasim": time.November, "kasım": time.November, "aralik": time.December, "aralık": time.December,
	"january": time.January, "february": time.February, "march": time.March,
	"april": time.April, "may": time.May, "june": time.June, "july": time.July,
	"august": time.August, "september": time.September, "october": time.October,
	"november": time.November, "december": time.December,
	"jan": time.January, "feb": time.February, "mar": time.March, "apr": time.April,
	"jun": time.June, "jul": time.July, "aug": time.August, "sep": time.September,
	"sept": time.September, "oct": time.October, "nov": time.November, "dec": time.December,
}

var dateLayouts = []string{
	"2006-01-02",
	"02.01.2006", "2.1.2006", "02.1.2006", "2.01.2006",
	"02/01/2006", "2/1/2006",
	"02-01-2006", "2-1-2006",
	"2006.01.02", "2006/01/02",
}

var rootKeys = []string{"meta", "donem", "oda_kontenjanlari", "fiyatlar", "release", "stop_sale", "opsiyonel", "cikarim_meta"}

var metaKeys = []string{"otel_adi", "acente_adi", "sozlesme_tipi", "sezon", "para_birimi", "kur_esasi", "yetkili_mahkeme", "imza_tarihi"}
var donemKeys = []string{"baslangic", "bitis", "alt_donemler"}
var odaKeys = []string{"oda_tipi", "adet", "aciklama"}
var fiyatKeys = []string{"oda_tipi", "tutar", "birim", "pansiyon", "alt_donem_ad"}
var releaseKeys = []string{"gun", "kapsam", "kaynak_ifade"}
var stopKeys = []string{"baslangic", "bitis", "kapsam", "bildirim_yontemi", "kaynak_ifade"}
var altDonemKeys = []string{"ad", "baslangic", "bitis"}
var cikarimKeys = []string{"guven", "kaynak_sayfa", "kaynak_madde"}
var opsiyonelKeys = []string{"cocuk_politikasi", "iptal_kosullari", "no_show", "overbooking", "odeme"}

var enumSets = map[string][]string{
	"sozlesme_tipi":    {"tamamen_garantili", "kismen_garantili", "garantisiz", "istege_bagli", "serbest_satis", "blok_rezervasyon", "blok_satin_alma", "belirtilmemis"},
	"sezon":            {"yaz", "kis", "yillik", "belirtilmemis"},
	"kur_esasi":        {"giris_gunu_tcmb", "cikis_gunu_tcmb", "sabit_kur", "belirtilmemis"},
	"pansiyon":         {"RO", "BB", "HB", "FB", "AI", "belirtilmemis"},
	"birim":            {"oda_gecelik", "kisi_gecelik"},
	"kapsam_release":   {"isim_listesi", "kontenjan_iadesi", "her_ikisi", "belirtilmemis"},
	"bildirim_yontemi": {"yazili", "faks", "eposta", "sistem", "belirtilmemis"},
	"sorumlu_taraf":    {"otel", "acente", "belirtilmemis"},
}

var enumAlias = map[string]string{
	"tamamen garantili": "tamamen_garantili",
	"kismen garantili":  "kismen_garantili",
	"kısmen_garantili":  "kismen_garantili",
	"yaz":               "yaz",
	"summer":            "yaz",
	"kis":               "kis",
	"kış":               "kis",
	"winter":            "kis",
	"yillik":            "yillik",
	"yıllık":            "yillik",
	"oda_gecelik":       "oda_gecelik",
	"oda/gece":          "oda_gecelik",
	"oda gece":          "oda_gecelik",
	"kisi_gecelik":      "kisi_gecelik",
	"kisi/gece":         "kisi_gecelik",
	"kisi gece":         "kisi_gecelik",
	"isim_listesi":      "isim_listesi",
	"isim listesi":      "isim_listesi",
	"kontenjan_iadesi":  "kontenjan_iadesi",
	"yazili":            "yazili",
	"yazılı":            "yazili",
	"eposta":            "eposta",
	"e-posta":           "eposta",
	"email":             "eposta",
	"ro":                "RO",
	"bb":                "BB",
	"hb":                "HB",
	"fb":                "FB",
	"ai":                "AI",
}

// Normalize şemaya zorlar ve yapılan düzeltmelerin listesini döner.
// Notlar alan yolu içerir, sözleşme değerleri içermez.
func Normalize(data map[string]any) (map[string]any, []string) {
	var notes []string
	out := cloneMap(data)
	if out == nil {
		out = map[string]any{}
	}
	liftStopSale(out, &notes)
	pruneRoot(out, &notes)
	fillRequired(out, &notes)
	normalizeWalk(out, "", &notes)
	log.Printf("json normalize edildi duzeltme=%d", len(notes))
	return out, notes
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func liftStopSale(root map[string]any, notes *[]string) {
	donem, ok := root["donem"].(map[string]any)
	if !ok {
		return
	}
	nested, ok := donem["stop_sale"]
	if !ok {
		return
	}
	delete(donem, "stop_sale")
	*notes = append(*notes, "stop_sale donem içinden üst düzeye taşındı")
	switch ns := nested.(type) {
	case []any:
		if existing, ok := root["stop_sale"].([]any); ok {
			root["stop_sale"] = append(existing, ns...)
		} else {
			root["stop_sale"] = ns
		}
	default:
		if _, ok := root["stop_sale"]; !ok {
			root["stop_sale"] = []any{nested}
		}
	}
}

func pruneRoot(root map[string]any, notes *[]string) {
	pruneObject(root, rootKeys, notes, "")
	if meta, ok := root["meta"].(map[string]any); ok {
		pruneObject(meta, metaKeys, notes, "meta")
	}
	if donem, ok := root["donem"].(map[string]any); ok {
		pruneObject(donem, donemKeys, notes, "donem")
		if alts, ok := donem["alt_donemler"].([]any); ok {
			for i, item := range alts {
				if m, ok := item.(map[string]any); ok {
					pruneObject(m, altDonemKeys, notes, "donem.alt_donemler["+strconv.Itoa(i)+"]")
				}
			}
		}
	}
	pruneArrayMaps(root, "oda_kontenjanlari", odaKeys, notes)
	pruneArrayMaps(root, "fiyatlar", fiyatKeys, notes)
	if rel, ok := root["release"].(map[string]any); ok {
		pruneObject(rel, releaseKeys, notes, "release")
	}
	pruneArrayMaps(root, "stop_sale", stopKeys, notes)
	if op, ok := root["opsiyonel"].(map[string]any); ok {
		pruneObject(op, opsiyonelKeys, notes, "opsiyonel")
	}
	if cm, ok := root["cikarim_meta"].(map[string]any); ok {
		for k, v := range cm {
			if m, ok := v.(map[string]any); ok {
				pruneObject(m, cikarimKeys, notes, "cikarim_meta."+k)
			}
		}
	}
}

func pruneArrayMaps(root map[string]any, key string, allowed []string, notes *[]string) {
	arr, ok := root[key].([]any)
	if !ok {
		return
	}
	for i, item := range arr {
		if m, ok := item.(map[string]any); ok {
			pruneObject(m, allowed, notes, key+"["+strconv.Itoa(i)+"]")
		}
	}
}

func pruneObject(obj map[string]any, allowed []string, notes *[]string, prefix string) {
	allow := make(map[string]struct{}, len(allowed))
	for _, k := range allowed {
		allow[k] = struct{}{}
	}
	for k := range obj {
		if _, ok := allow[k]; ok {
			continue
		}
		delete(obj, k)
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		*notes = append(*notes, "bilinmeyen alan atıldı: "+path)
	}
}

func fillRequired(root map[string]any, notes *[]string) {
	if !isMap(root["donem"]) {
		root["donem"] = map[string]any{"baslangic": nil, "bitis": nil}
		*notes = append(*notes, "zorunlu alan dolduruldu: donem")
	} else {
		d := root["donem"].(map[string]any)
		ensureKey(d, "baslangic", nil, notes, "donem.baslangic")
		ensureKey(d, "bitis", nil, notes, "donem.bitis")
	}
	if !isSlice(root["oda_kontenjanlari"]) {
		root["oda_kontenjanlari"] = []any{}
		*notes = append(*notes, "zorunlu alan dolduruldu: oda_kontenjanlari")
	}
	if !isSlice(root["fiyatlar"]) {
		root["fiyatlar"] = []any{}
		*notes = append(*notes, "zorunlu alan dolduruldu: fiyatlar")
	}
	if !isMap(root["release"]) {
		root["release"] = map[string]any{"gun": nil}
		*notes = append(*notes, "zorunlu alan dolduruldu: release")
	} else {
		ensureKey(root["release"].(map[string]any), "gun", nil, notes, "release.gun")
	}
	if !isSlice(root["stop_sale"]) {
		root["stop_sale"] = []any{}
		*notes = append(*notes, "zorunlu alan dolduruldu: stop_sale")
	}
}

func ensureKey(m map[string]any, key string, empty any, notes *[]string, path string) {
	if _, ok := m[key]; !ok {
		m[key] = empty
		*notes = append(*notes, "zorunlu alan dolduruldu: "+path)
	}
}

func isMap(v any) bool {
	_, ok := v.(map[string]any)
	return ok
}

func isSlice(v any) bool {
	_, ok := v.([]any)
	return ok
}

func normalizeWalk(root map[string]any, _ string, notes *[]string) {
	if meta, ok := root["meta"].(map[string]any); ok {
		applyEnum(meta, "sozlesme_tipi", "sozlesme_tipi", "meta.sozlesme_tipi", notes)
		applyEnum(meta, "sezon", "sezon", "meta.sezon", notes)
		applyEnum(meta, "kur_esasi", "kur_esasi", "meta.kur_esasi", notes)
		applyDate(meta, "imza_tarihi", "meta.imza_tarihi", notes)
		if s, ok := meta["para_birimi"].(string); ok {
			code := strings.ToUpper(strings.TrimSpace(s))
			if code != s && len(code) == 3 {
				meta["para_birimi"] = code
				*notes = append(*notes, "para birimi normalize edildi: meta.para_birimi")
			}
		}
	}
	if donem, ok := root["donem"].(map[string]any); ok {
		applyDate(donem, "baslangic", "donem.baslangic", notes)
		applyDate(donem, "bitis", "donem.bitis", notes)
		if alts, ok := donem["alt_donemler"].([]any); ok {
			for i, item := range alts {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				p := "donem.alt_donemler[" + strconv.Itoa(i) + "]"
				applyDate(m, "baslangic", p+".baslangic", notes)
				applyDate(m, "bitis", p+".bitis", notes)
			}
		}
	}
	if arr, ok := root["oda_kontenjanlari"].([]any); ok {
		for i, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			applyInt(m, "adet", "oda_kontenjanlari["+strconv.Itoa(i)+"].adet", notes)
		}
	}
	if arr, ok := root["fiyatlar"].([]any); ok {
		for i, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			p := "fiyatlar[" + strconv.Itoa(i) + "]"
			applyNumber(m, "tutar", p+".tutar", notes)
			applyEnum(m, "birim", "birim", p+".birim", notes)
			applyEnum(m, "pansiyon", "pansiyon", p+".pansiyon", notes)
		}
	}
	if rel, ok := root["release"].(map[string]any); ok {
		applyInt(rel, "gun", "release.gun", notes)
		applyEnum(rel, "kapsam", "kapsam_release", "release.kapsam", notes)
	}
	if arr, ok := root["stop_sale"].([]any); ok {
		for i, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			p := "stop_sale[" + strconv.Itoa(i) + "]"
			applyDate(m, "baslangic", p+".baslangic", notes)
			applyDate(m, "bitis", p+".bitis", notes)
			applyEnum(m, "bildirim_yontemi", "bildirim_yontemi", p+".bildirim_yontemi", notes)
		}
	}
	if op, ok := root["opsiyonel"].(map[string]any); ok {
		if ns, ok := op["no_show"].(map[string]any); ok {
			applyEnum(ns, "sorumlu_taraf", "sorumlu_taraf", "opsiyonel.no_show.sorumlu_taraf", notes)
		}
		if ov, ok := op["overbooking"].(map[string]any); ok {
			applyEnum(ov, "sorumlu_taraf", "sorumlu_taraf", "opsiyonel.overbooking.sorumlu_taraf", notes)
		}
	}
}

func applyEnum(obj map[string]any, key, setName, path string, notes *[]string) {
	v, ok := obj[key]
	if !ok || v == nil {
		return
	}
	s, ok := v.(string)
	if !ok {
		obj[key] = "belirtilmemis"
		*notes = append(*notes, "enum normalize edildi: "+path)
		return
	}
	canon := canonicalEnum(s, setName)
	if canon != s {
		obj[key] = canon
		*notes = append(*notes, "enum normalize edildi: "+path)
	}
}

func canonicalEnum(raw, setName string) string {
	allowed := enumSets[setName]
	for _, a := range allowed {
		if raw == a {
			return a
		}
	}
	folded := foldTR(raw)
	folded = strings.ReplaceAll(folded, "-", "_")
	folded = strings.Join(strings.Fields(folded), " ")
	if alias, ok := enumAlias[folded]; ok {
		folded = foldTR(alias)
	}
	underscored := strings.ReplaceAll(folded, " ", "_")
	for _, a := range allowed {
		if foldTR(a) == underscored || a == underscored {
			return a
		}
	}
	if setName == "pansiyon" {
		u := strings.ToUpper(strings.TrimSpace(raw))
		for _, a := range allowed {
			if a == u {
				return a
			}
		}
	}
	return "belirtilmemis"
}

func foldTR(s string) string {
	r := strings.NewReplacer(
		"İ", "i", "I", "i", "ı", "i",
		"Ş", "s", "ş", "s",
		"Ğ", "g", "ğ", "g",
		"Ü", "u", "ü", "u",
		"Ö", "o", "ö", "o",
		"Ç", "c", "ç", "c",
	)
	return strings.ToLower(r.Replace(strings.TrimSpace(s)))
}

func applyDate(obj map[string]any, key, path string, notes *[]string) {
	v, ok := obj[key]
	if !ok || v == nil {
		return
	}
	s, ok := v.(string)
	if !ok {
		return
	}
	iso, ok := parseDate(s)
	if !ok || iso == s {
		return
	}
	obj[key] = iso
	*notes = append(*notes, "tarih ISO-8601 yapıldı: "+path)
}

func parseDate(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02"), true
		}
	}
	m := namedDate.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	day, _ := strconv.Atoi(m[1])
	year, _ := strconv.Atoi(m[3])
	monthName := foldTR(m[2])
	month, ok := trMonths[monthName]
	if !ok {
		return "", false
	}
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if t.Day() != day || t.Month() != month || t.Year() != year {
		return "", false
	}
	return t.Format("2006-01-02"), true
}

func applyInt(obj map[string]any, key, path string, notes *[]string) {
	v, ok := obj[key]
	if !ok || v == nil {
		return
	}
	n, ok := toInt(v)
	if !ok {
		if s, isStr := v.(string); isStr {
			if parsed, ok := parseLooseNumber(s); ok {
				if i, ok := toInt(parsed); ok {
					obj[key] = i
					*notes = append(*notes, "sayı temizlendi: "+path)
					return
				}
			}
		}
		return
	}
	if _, isInt := v.(int); isInt {
		return
	}
	obj[key] = n
	if _, isStr := v.(string); isStr {
		*notes = append(*notes, "sayı temizlendi: "+path)
	}
}

func applyNumber(obj map[string]any, key, path string, notes *[]string) {
	v, ok := obj[key]
	if !ok || v == nil {
		return
	}
	if s, isStr := v.(string); isStr {
		if parsed, ok := parseLooseNumber(s); ok {
			obj[key] = parsed
			*notes = append(*notes, "sayı temizlendi: "+path)
			return
		}
	}
	if n, ok := toFloat(v); ok {
		switch v.(type) {
		case float64, int, int64:
			_ = n
			return
		default:
			obj[key] = n
			*notes = append(*notes, "sayı temizlendi: "+path)
		}
	}
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		if n == float64(int(n)) {
			return int(n), true
		}
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return int(i), true
		}
	}
	return 0, false
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func parseLooseNumber(s string) (any, bool) {
	s = strings.TrimSpace(s)
	s = currencyPrefix.ReplaceAllString(s, "")
	s = currencySuffix.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
	if s == "" {
		return nil, false
	}
	if m := euThousands.FindStringSubmatch(s); m != nil {
		digits := strings.ReplaceAll(m[1], ".", "")
		if m[2] != "" {
			f, err := strconv.ParseFloat(digits+"."+m[2], 64)
			return f, err == nil
		}
		i, err := strconv.Atoi(digits)
		return i, err == nil
	}
	if m := usThousands.FindStringSubmatch(s); m != nil {
		digits := strings.ReplaceAll(m[1], ",", "")
		if m[2] != "" {
			f, err := strconv.ParseFloat(digits+"."+m[2], 64)
			return f, err == nil
		}
		i, err := strconv.Atoi(digits)
		return i, err == nil
	}
	if m := euDecimal.FindStringSubmatch(s); m != nil {
		f, err := strconv.ParseFloat(m[1]+"."+m[2], 64)
		return f, err == nil
	}
	if i, err := strconv.Atoi(s); err == nil {
		return i, true
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, true
	}
	return nil, false
}
