package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/oz-fatma/kontrata/backend/internal/extract"
	"github.com/oz-fatma/kontrata/backend/internal/llm"
	"github.com/oz-fatma/kontrata/backend/internal/mask"
)

const maxChunkCorrectionRounds = 1

type chunkOutcome struct {
	data    map[string]any
	retries int
	genErr  error
}

func (r *Reader) extractChunked(ctx context.Context, pages []string) (*ExtractResult, error) {
	start := time.Now()
	out := &ExtractResult{Data: map[string]any{}}
	if r == nil || r.LLM == nil {
		out.SchemaErrors = []string{"model yapılandırılmadı"}
		out.Duration = time.Since(start)
		return out, nil
	}

	masked := mask.Apply(joinPages(pages))
	log.Printf("cikarma chunked basladi alan=%d", masked.Count)

	outcomes := make([]chunkOutcome, len(chunkSpecs))
	var wg sync.WaitGroup
	for i, spec := range chunkSpecs {
		wg.Add(1)
		go func(idx int, sp chunkSpec) {
			defer wg.Done()
			outcomes[idx] = r.fetchChunk(ctx, sp, masked.Text)
		}(i, spec)
	}
	wg.Wait()

	merged := map[string]any{}
	maxRetries := 0
	for i, o := range outcomes {
		if o.retries > maxRetries {
			maxRetries = o.retries
		}
		if o.genErr != nil {
			log.Printf("cikarma parca=%s model hatasi: %v", chunkSpecs[i].agent, o.genErr)
			continue
		}
		if o.data == nil {
			log.Printf("cikarma parca=%s json cozulemedi", chunkSpecs[i].agent)
			continue
		}
		mergeMaps(merged, o.data)
	}

	data, repairs := extract.Normalize(merged)
	errs := extract.Validate(data)
	attachMeta(data, repairs, pages)
	out.Data = data
	out.Repairs = repairs
	out.SchemaErrors = errs
	out.Meta = BuildExtractionMeta(data, repairs, pages)
	out.RetryCount = maxRetries
	out.Duration = time.Since(start)
	return out, nil
}

func (r *Reader) fetchChunk(ctx context.Context, spec chunkSpec, contract string) chunkOutcome {
	client := llm.LimitTokens(r.LLM, spec.maxTokens)
	callCtx := withChunkAgent(ctx, spec.agent)

	raw, err := client.Generate(callCtx, spec.prompt, contract)
	if err != nil {
		return chunkOutcome{genErr: err}
	}
	r.dumpExchange(contract, raw)

	var best map[string]any
	parsed, parseErr := repairChunk(raw)
	if parseErr == nil {
		parsed = prepareChunk(spec, parsed, contract)
		if chunkUsable(spec, parsed, contract) {
			best = parsed
		} else {
			parseErr = errChunkUnusable
		}
	}
	retries := 0
	for round := 1; round <= maxChunkCorrectionRounds && parseErr != nil; round++ {
		retries = round
		log.Printf("cikarma parca=%s duzeltme tur=%d", spec.agent, round)
		corr := chunkCorrectionPrompt(contract, spec, parseErr)
		raw2, genErr := client.Generate(callCtx, spec.prompt, corr)
		if genErr != nil {
			if best != nil {
				return chunkOutcome{data: best, retries: retries}
			}
			return chunkOutcome{retries: retries, genErr: genErr}
		}
		r.dumpExchange(corr, raw2)
		parsed, parseErr = repairChunk(raw2)
		if parseErr == nil {
			parsed = prepareChunk(spec, parsed, contract)
			if chunkUsable(spec, parsed, contract) {
				best = parsed
			} else {
				parseErr = errChunkUnusable
			}
		}
	}
	if parseErr != nil {
		// Düzeltme bozulursa ilk kullanılabilir çıktıyı koru (boş parça > şema HATA).
		if best != nil {
			return chunkOutcome{data: best, retries: retries}
		}
		return chunkOutcome{retries: retries}
	}
	return chunkOutcome{data: parsed, retries: retries}
}

func prepareChunk(spec chunkSpec, data map[string]any, contract string) map[string]any {
	if data == nil {
		return nil
	}
	switch spec.agent {
	case llm.AgentReaderChunkA:
		return sanitizeChunkA(data, contract)
	case llm.AgentReaderChunkB:
		return sanitizeChunkB(data, contract)
	case llm.AgentReaderChunkC:
		return sanitizeChunkC(data, contract)
	case llm.AgentReaderChunkD:
		return sanitizeChunkD(data, contract)
	default:
		return data
	}
}

var errChunkUnusable = errors.New("parça beklenen alanları içermiyor")

func chunkUsable(spec chunkSpec, data map[string]any, contract string) bool {
	if data == nil {
		return false
	}
	switch spec.agent {
	case llm.AgentReaderChunkA:
		oda, _ := data["oda_kontenjanlari"].([]any)
		return len(oda) > 0
	case llm.AgentReaderChunkB:
		// Eksik satır için agresif tooFew retry küçük modelde şema HATA'sına yol açtı; kaldırıldı.
		return fiyatlarUsable(data["fiyatlar"])
	case llm.AgentReaderChunkC:
		_, ok := data["release"].(map[string]any)
		if ok {
			return true
		}
		arr, ok := data["release"].([]any)
		return ok && len(arr) > 0
	case llm.AgentReaderChunkD:
		meta, _ := data["meta"].(map[string]any)
		return len(meta) > 0
	default:
		return len(data) > 0
	}
}

func fiyatlarUsable(v any) bool {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return false
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if _, hasTip := m["oda_tipi"]; hasTip {
			return true
		}
		if _, hasTutar := m["tutar"]; hasTutar {
			return true
		}
	}
	return false
}

// fiyatlarTooFew fiyat tablosunda birden fazla satır varken modelin tek satır dönmesini yakalar.
func fiyatlarTooFew(v any, contract string) bool {
	arr, ok := v.([]any)
	if !ok {
		return true
	}
	hint := countRateRowHints(contract)
	if hint < 2 {
		return false
	}
	return len(arr) < hint
}

func countRateRowHints(contract string) int {
	section := fiyatSection(contract)
	if section == "" {
		section = strings.ToLower(contract)
	}
	lines := strings.Split(section, "\n")
	n := 0
	for _, line := range lines {
		low := strings.ToLower(strings.TrimSpace(line))
		if low == "" {
			continue
		}
		if !lineHasRoomTypeHint(low) {
			continue
		}
		if _, ok := firstIntInChunkText(low); ok {
			n++
		}
	}
	return n
}

func lineHasRoomTypeHint(low string) bool {
	hints := []string{
		"standart", "standard", "normal oda",
		"suit", "suite", "penthouse",
		"aile", "family",
		"deluxe", "balayi", "balayı", "honeymoon",
		"engelli", "özürlü", "accessible",
		"tek kişilik", "iki kişilik", "üç kişilik", "tek kisilik", "iki kisilik", "uc kisilik",
	}
	for _, h := range hints {
		if strings.Contains(low, h) {
			return true
		}
	}
	return false
}

func fiyatSection(contract string) string {
	return sectionBetween(contract, []string{
		"oda fiyat", "gecelik ücret", "gecelik ucret", "rates", "fiyatlar",
		"madde 8", "article 4", "4. fiyat",
	}, []string{
		"madde 5", "article 5", "madde 6", "article 6",
		"ödeme", "odeme", "payment", "iptal", "cancel",
		"stop-sale", "stop sale", "satış durdur", "yetkili mahkeme", "jurisdiction",
		"çocuk", "cocuk", "child",
	})
}

func kontenjanSection(contract string) string {
	return sectionBetween(contract, []string{
		"oda kontenjan", "room allotment", "oda tahsis",
		"madde 2", "article 2", "2. oda kontenjan",
	}, []string{
		"madde 3", "article 3", "madde 4", "article 4",
		"oda fiyat", "gecelik ücret", "gecelik ucret", "rates", "fiyatlar",
		"name list", "isim listesi", "release",
		"ödeme", "odeme", "payment", "3. release",
	})
}

func sectionBetween(contract string, starts, ends []string) string {
	low := strings.ToLower(contract)
	start := -1
	for _, m := range starts {
		if i := strings.Index(low, m); i >= 0 {
			if start < 0 || i < start {
				start = i
			}
		}
	}
	if start < 0 {
		return ""
	}
	rest := low[start:]
	end := len(rest)
	from := min(40, len(rest))
	for _, m := range ends {
		if i := strings.Index(rest[from:], m); i >= 0 {
			pos := from + i
			if pos < end {
				end = pos
			}
		}
	}
	return rest[:end]
}

func chunkCorrectionPrompt(contract string, spec chunkSpec, cause error) string {
	var b strings.Builder
	b.WriteString(contract)
	b.WriteString("\n\nÖnceki çıktı geçersiz.")
	if errors.Is(cause, errChunkUnusable) {
		switch spec.agent {
		case llm.AgentReaderChunkA:
			b.WriteString(" SADECE donem ve oda_kontenjanlari içeren tek JSON nesnesi üret.")
		case llm.AgentReaderChunkB:
			b.WriteString(" SADECE {\"fiyatlar\":[...]} biçiminde tek JSON nesnesi üret. Fiyat tablosundaki HER oda satırını yaz; tek satır bırakma. Kod veya açıklama YAZMA.")
		case llm.AgentReaderChunkC:
			b.WriteString(" SADECE release ve stop_sale içeren tek JSON nesnesi üret. stop_sale yoksa []. Sezon adından stop_sale uydurma.")
		case llm.AgentReaderChunkD:
			b.WriteString(" SADECE {\"meta\":{...}} biçiminde tek JSON nesnesi üret. Önce otel_adi, acente_adi, para_birimi yaz. sozlesme_tipi/sezon/belirtilmemis uydurma.")
		}
	} else {
		b.WriteString(" JSON olarak çözülemedi.")
	}
	b.WriteString("\nDüzeltilmiş tek JSON nesnesi üret. Açıklama yazma.")
	return b.String()
}

func sanitizeChunkA(data map[string]any, contract string) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if oda, ok := data["oda_kontenjanlari"]; ok {
		out["oda_kontenjanlari"] = sanitizeOdaKontenjanlari(oda, contract)
	}

	donem := map[string]any{}
	if d, ok := data["donem"].(map[string]any); ok {
		for _, k := range []string{"baslangic", "bitis", "alt_donemler"} {
			if v, ok := d[k]; ok {
				donem[k] = v
			}
		}
	}
	if alt, ok := data["alt_donemler"]; ok && donem["alt_donemler"] == nil {
		donem["alt_donemler"] = alt
	}
	donem["alt_donemler"] = validAltDonemler(donem["alt_donemler"])
	if len(donem) > 0 {
		// Ters tarihleri sessizce swap etme — denetçi R1 yakalar (coral-bozuk).
		out["donem"] = donem
	}
	return out
}

func sanitizeChunkB(data map[string]any, contract string) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	arr, ok := data["fiyatlar"].([]any)
	if !ok {
		return data
	}
	low := strings.ToLower(contract)
	defaultBirim := "oda_gecelik"
	if strings.Contains(low, "kişi başı") || strings.Contains(low, "kisi basi") ||
		strings.Contains(low, "per person") || strings.Contains(low, "kişi başı gecelik") {
		defaultBirim = "kisi_gecelik"
	}
	defaultPansiyon := ""
	switch {
	case strings.Contains(low, " her şey dahil") || strings.Contains(low, "her sey dahil") ||
		strings.Contains(low, "(ai)") || strings.Contains(low, " all inclusive") ||
		regexp.MustCompile(`(?i)\bai\b`).MatchString(contract):
		defaultPansiyon = "AI"
	case strings.Contains(low, "bed and breakfast") || strings.Contains(low, " kahvaltı") ||
		regexp.MustCompile(`(?i)\bbb\b`).MatchString(contract):
		defaultPansiyon = "BB"
	}

	out := make([]any, 0, len(arr))
	seen := map[string]struct{}{}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		cp := map[string]any{}
		tip := chunkAOdaTipi(m["oda_tipi"])
		if tip == "" {
			continue
		}
		cp["oda_tipi"] = tip
		if tutar, ok := m["tutar"]; ok && tutar != nil {
			cp["tutar"] = tutar
		} else {
			continue
		}
		cp["birim"] = sanitizeFiyatBirim(m["birim"], defaultBirim)
		if p := sanitizeFiyatPansiyon(m["pansiyon"], defaultPansiyon); p != "" {
			cp["pansiyon"] = p
		}
		if ad, ok := m["alt_donem_ad"].(string); ok {
			ad = strings.TrimSpace(ad)
			if ad != "" && !strings.EqualFold(ad, "null") {
				cp["alt_donem_ad"] = ad
			}
		}
		key := tip + "|" + fmt.Sprint(cp["tutar"]) + "|" + fmt.Sprint(cp["alt_donem_ad"]) + "|" + fmt.Sprint(cp["pansiyon"])
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, cp)
	}
	out = enrichFiyatlarFromContract(out, seen, contract, defaultBirim, defaultPansiyon)
	return map[string]any{"fiyatlar": out}
}

func sanitizeFiyatBirim(v any, fallback string) string {
	// Sözleşme metninden çıkan birim öncelikli (model sıkça karıştırır).
	if fallback != "" {
		return fallback
	}
	s, _ := v.(string)
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "_")
	switch {
	case s == "oda_gecelik" || strings.Contains(s, "oda_gecelik") || strings.Contains(s, "per_room") || strings.Contains(s, "room_night"):
		return "oda_gecelik"
	case s == "kisi_gecelik" || strings.Contains(s, "kisi") || strings.Contains(s, "person"):
		return "kisi_gecelik"
	default:
		return "oda_gecelik"
	}
}

func sanitizeFiyatPansiyon(v any, fallback string) string {
	s, _ := v.(string)
	s = strings.TrimSpace(s)
	u := strings.ToUpper(s)
	switch u {
	case "RO", "BB", "HB", "FB", "AI":
		// Metinde net pansiyon varsa (AI/BB) modelin RO uydurmasını ez.
		if fallback != "" && fallback != u && (u == "RO" || u == "BELIRTILMEMIS") {
			return fallback
		}
		return u
	}
	low := strings.ToLower(s)
	switch {
	case strings.Contains(low, "all inclusive") || strings.Contains(low, "her şey") || low == "ai":
		return "AI"
	case strings.Contains(low, "bed and breakfast") || low == "bb":
		return "BB"
	case low == "belirtilmemis" || s == "" || strings.EqualFold(s, "null"):
		if fallback != "" {
			return fallback
		}
		return "belirtilmemis"
	default:
		if fallback != "" {
			return fallback
		}
		return "belirtilmemis"
	}
}

func sanitizeChunkC(data map[string]any, contract string) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if rel, ok := data["release"]; ok {
		out["release"] = coerceReleaseObject(rel)
	}
	out["stop_sale"] = sanitizeStopSale(data["stop_sale"], contract)
	return out
}

func coerceReleaseObject(v any) any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	if arr, ok := v.([]any); ok && len(arr) > 0 {
		if m, ok := arr[0].(map[string]any); ok {
			return m
		}
	}
	return v
}

func sanitizeStopSale(v any, contract string) []any {
	low := strings.ToLower(contract)
	if contract != "" && !contractMentionsStopSale(low) {
		return []any{}
	}
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		bas, _ := m["baslangic"].(string)
		bit, _ := m["bitis"].(string)
		bas = strings.TrimSpace(bas)
		bit = strings.TrimSpace(bit)
		if bas == "" && bit == "" {
			continue
		}
		if contract != "" && !stopSaleDatesInContract(low, bas, bit) {
			continue
		}
		cp := map[string]any{}
		if bas != "" {
			cp["baslangic"] = bas
		}
		if bit != "" {
			cp["bitis"] = bit
		}
		if k, ok := m["kapsam"].(string); ok {
			if nk := normalizeStopSaleKapsam(k); nk != "" {
				cp["kapsam"] = nk
			}
		}
		out = append(out, cp)
	}
	return out
}

func contractMentionsStopSale(low string) bool {
	markers := []string{
		"stop-sale", "stop sale", "stopsale",
		"satış durdur", "satis durdur", "satışı durdur", "satisi durdur",
		"satışa kapat", "satisa kapat",
	}
	for _, m := range markers {
		if strings.Contains(low, m) {
			return true
		}
	}
	return false
}

func stopSaleDatesInContract(low, bas, bit string) bool {
	okBas := bas == "" || dateMentionedInContract(low, bas)
	okBit := bit == "" || dateMentionedInContract(low, bit)
	return okBas && okBit
}

func dateMentionedInContract(low, isoOrRaw string) bool {
	s := strings.TrimSpace(isoOrRaw)
	if s == "" {
		return false
	}
	// ISO 2026-07-10
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' {
		y, m, d := s[0:4], s[5:7], s[8:10]
		forms := []string{
			s, // 2026-07-10
			d + "." + m + "." + y,           // 10.07.2026
			strings.TrimLeft(d, "0") + "." + strings.TrimLeft(m, "0") + "." + y,
			d + "/" + m + "/" + y,
		}
		for _, f := range forms {
			if strings.Contains(low, strings.ToLower(f)) {
				return true
			}
		}
		return false
	}
	return strings.Contains(low, strings.ToLower(s))
}

func normalizeStopSaleKapsam(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	low := strings.ToLower(s)
	low = strings.ReplaceAll(low, "_", " ")
	switch {
	case strings.Contains(low, "tüm oda") || strings.Contains(low, "tum oda") ||
		strings.Contains(low, "tum odai") || strings.Contains(low, "all room") ||
		strings.Contains(low, "tüm tesis") || strings.Contains(low, "tum tesis"):
		return "tüm oda tipleri"
	default:
		if tip := chunkAOdaTipi(s); tip != "" {
			return tip
		}
		return s
	}
}

func enrichFiyatlarFromContract(out []any, seen map[string]struct{}, contract, defaultBirim, defaultPansiyon string) []any {
	section := fiyatSection(contract)
	if section == "" {
		return out
	}
	if defaultBirim == "" {
		defaultBirim = "oda_gecelik"
	}
	if defaultPansiyon == "" {
		defaultPansiyon = "belirtilmemis"
	}
	for _, line := range strings.Split(section, "\n") {
		low := strings.ToLower(strings.TrimSpace(line))
		if low == "" || !lineHasRoomTypeHint(low) {
			continue
		}
		tip := tipFromRateLine(low)
		if tip == "" {
			continue
		}
		n, ok := firstIntInChunkText(low)
		if !ok || n <= 0 {
			continue
		}
		// Fiyat satırındaki kontenjan adetlerini (150, 40) atlama: rates'te genelde 2+ basamaklı ondalık/ücret.
		// Kontenjan bölümündeki büyük adetler rates satırında tek başına nadiren gelir; yine de
		// junior suite / penthouse / suit fiyatlarını mutlaka al.
		cp := map[string]any{
			"oda_tipi": tip,
			"tutar":    float64(n),
			"birim":    defaultBirim,
			"pansiyon": defaultPansiyon,
		}
		key := tip + "|" + fmt.Sprint(cp["tutar"]) + "||" + defaultPansiyon
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, cp)
	}
	return out
}

func tipFromRateLine(low string) string {
	// Daha spesifik eşleşmeler önce.
	checks := []struct {
		sub string
		tip string
	}{
		{"junior suite", "suit"},
		{"penthouse", "suit"},
		{"honeymoon", "balayi"},
		{"balayı", "balayi"},
		{"balayi", "balayi"},
		{"özürlü", "engelli"},
		{"ozurlu", "engelli"},
		{"accessible", "engelli"},
		{"engelli", "engelli"},
		{"family", "aile"},
		{"aile", "aile"},
		{"deluxe", "deluxe"},
		{"suite", "suit"},
		{"suit", "suit"},
		{"standard", "standart"},
		{"standart", "standart"},
		{"tek kişilik", "standart"},
		{"iki kişilik", "standart"},
		{"üç kişilik", "standart"},
		{"tek kisilik", "standart"},
		{"iki kisilik", "standart"},
		{"uc kisilik", "standart"},
		{"normal oda", "standart"},
	}
	for _, c := range checks {
		if strings.Contains(low, c.sub) {
			return c.tip
		}
	}
	return ""
}

func sanitizeChunkD(data map[string]any, contract string) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	meta := sanitizeChunkMeta(data["meta"], contract)
	if len(meta) == 0 {
		// Model meta'yı üst düzey alanlar olarak da yazmış olabilir.
		meta = sanitizeChunkMeta(data, contract)
		if len(meta) == 0 {
			return map[string]any{}
		}
	}
	return map[string]any{"meta": meta}
}

var chunkMetaKeys = []string{
	"otel_adi", "acente_adi", "para_birimi", "kur_esasi",
	"yetkili_mahkeme", "sozlesme_tipi", "sezon",
}

var chunkAMetaKeys = chunkMetaKeys // canlı test uyumluluğu

var chunkAParaBirimi = map[string]struct{}{
	"EUR": {}, "GBP": {}, "USD": {}, "TRY": {},
}

var chunkAKurEsasi = map[string]struct{}{
	"giris_gunu_tcmb": {}, "cikis_gunu_tcmb": {}, "sabit_kur": {}, "belirtilmemis": {},
}

var chunkASozlesmeTipi = map[string]struct{}{
	"tamamen_garantili": {}, "kismen_garantili": {}, "garantisiz": {}, "istege_bagli": {},
	"serbest_satis": {}, "blok_rezervasyon": {}, "blok_satin_alma": {}, "belirtilmemis": {},
}

var chunkASezon = map[string]struct{}{
	"yaz": {}, "kis": {}, "yillik": {}, "belirtilmemis": {},
}

func sanitizeChunkMeta(v any, contract string) map[string]any {
	src, ok := v.(map[string]any)
	if !ok {
		src = map[string]any{}
	}
	low := strings.ToLower(contract)
	out := map[string]any{}
	for _, k := range chunkMetaKeys {
		raw, ok := src[k]
		if !ok || raw == nil {
			continue
		}
		switch k {
		case "otel_adi", "acente_adi":
			s, ok := raw.(string)
			s = strings.TrimSpace(s)
			if !ok || s == "" {
				continue
			}
			// Metinde geçmeyen taraf adını uydurma say.
			if contract != "" && !partyNameInContract(low, s) {
				continue
			}
			out[k] = s
		case "para_birimi":
			s, ok := raw.(string)
			if !ok {
				continue
			}
			code := strings.ToUpper(strings.TrimSpace(s))
			if _, ok := chunkAParaBirimi[code]; !ok {
				continue
			}
			// Metinde geçmeyen para birimini uydurma say.
			if contract != "" && !strings.Contains(low, strings.ToLower(code)) &&
				!paraBirimiMentioned(low, code) {
				continue
			}
			out[k] = code
		case "kur_esasi":
			if s, ok := chunkAEnumString(raw, chunkAKurEsasi); ok {
				if s == "belirtilmemis" {
					continue
				}
				if contract == "" || kurEsasiMentioned(low, s) {
					out[k] = s
				}
			}
		case "sozlesme_tipi":
			if s, ok := chunkAEnumString(raw, chunkASozlesmeTipi); ok {
				if s == "belirtilmemis" {
					continue
				}
				if contract == "" || sozlesmeTipiMentioned(low, s) {
					out[k] = s
				}
			}
		case "sezon":
			if s, ok := chunkAEnumString(raw, chunkASezon); ok {
				if s == "belirtilmemis" {
					continue
				}
				if contract == "" || sezonMentioned(low, s) {
					out[k] = s
				}
			}
		case "yetkili_mahkeme":
			if s, ok := sanitizeYetkiliMahkeme(raw); ok {
				if contract == "" || strings.Contains(low, strings.ToLower(s)) {
					out[k] = s
				}
			}
		}
	}

	enrichMetaFromContract(out, contract, low)

	if len(out) == 0 {
		return nil
	}
	return out
}

func enrichMetaFromContract(out map[string]any, contract, low string) {
	if contract == "" {
		return
	}
	if _, ok := out["para_birimi"]; !ok {
		if code, ok := inferParaBirimi(low); ok {
			out["para_birimi"] = code
		}
	}
	if _, ok := out["kur_esasi"]; !ok {
		if k, ok := inferKurEsasi(low); ok {
			out["kur_esasi"] = k
		}
	}
	otel, _ := out["otel_adi"].(string)
	acente, _ := out["acente_adi"].(string)
	if otel != "" && acente != "" {
		return
	}
	infOtel, infAcente, ok := inferParties(contract)
	if !ok {
		return
	}
	if otel == "" && infOtel != "" {
		out["otel_adi"] = infOtel
	}
	if acente == "" && infAcente != "" {
		out["acente_adi"] = infAcente
	}
}

func inferKurEsasi(low string) (string, bool) {
	for _, k := range []string{"giris_gunu_tcmb", "cikis_gunu_tcmb", "sabit_kur"} {
		if kurEsasiMentioned(low, k) {
			return k, true
		}
	}
	return "", false
}

func partyNameInContract(lowContract, name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	if strings.Contains(lowContract, n) {
		return true
	}
	// Noktalama farkı: "A.Ş." vs "A.Ş"
	compact := strings.Map(func(r rune) rune {
		switch r {
		case '.', ',', ';', ':', '·', '(', ')', '"', '\'':
			return -1
		default:
			return r
		}
	}, n)
	compact = strings.Join(strings.Fields(compact), " ")
	if compact == "" || len([]rune(compact)) < 4 {
		return false
	}
	lowCompact := strings.Map(func(r rune) rune {
		switch r {
		case '.', ',', ';', ':', '·', '(', ')', '"', '\'':
			return -1
		default:
			return r
		}
	}, lowContract)
	lowCompact = strings.Join(strings.Fields(lowCompact), " ")
	return strings.Contains(lowCompact, compact)
}

func inferParaBirimi(low string) (string, bool) {
	var found []string
	for _, code := range []string{"EUR", "GBP", "USD", "TRY"} {
		if strings.Contains(low, strings.ToLower(code)) || paraBirimiMentioned(low, code) {
			found = append(found, code)
		}
	}
	if len(found) == 1 {
		return found[0], true
	}
	return "", false
}

var (
	reTaraflarLine = regexp.MustCompile(`(?is)Taraflar\s*:\s*(.+?)\s*\((?:Operatör|Operator|Acente|Tour\s*Operator)\)\s*[—\-–]\s*(.+?)\s*\((?:Tesis|Otel|Hotel|Property)\)`)
	reIleArasinda  = regexp.MustCompile(`(?is)([^\n.]{3,80}?)\s+ile\s+([^\n.]{3,80}?)\s+arasında`)
)

func inferParties(contract string) (otel, acente string, ok bool) {
	if m := reTaraflarLine.FindStringSubmatch(contract); len(m) == 3 {
		acente = cleanPartyName(m[1])
		otel = cleanPartyName(m[2])
		if otel != "" && acente != "" {
			return otel, acente, true
		}
	}
	if m := reIleArasinda.FindStringSubmatch(contract); len(m) == 3 {
		acente = cleanPartyName(m[1])
		otel = cleanPartyName(m[2])
		// Başlık satırlarını ele: "HİZMET SÖZLEŞMESİ\nSide Turizm..."
		if i := strings.LastIndex(acente, "\n"); i >= 0 {
			acente = cleanPartyName(acente[i+1:])
		}
		if otel != "" && acente != "" {
			return otel, acente, true
		}
	}
	return "", "", false
}

func cleanPartyName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, " \t.—–·-")
	// Çok satırlı yakalamada son satırı al.
	if lines := strings.Split(s, "\n"); len(lines) > 1 {
		s = strings.TrimSpace(lines[len(lines)-1])
	}
	s = strings.TrimSpace(s)
	if len([]rune(s)) < 3 || len([]rune(s)) > 80 {
		return ""
	}
	return s
}

func sozlesmeTipiMentioned(low, tip string) bool {
	switch tip {
	case "tamamen_garantili":
		return strings.Contains(low, "tamamen garantili") ||
			strings.Contains(low, "full guarantee") ||
			strings.Contains(low, "fully guaranteed")
	case "kismen_garantili":
		return strings.Contains(low, "kısmen garantili") ||
			strings.Contains(low, "kismen garantili") ||
			strings.Contains(low, "partial guarantee") ||
			strings.Contains(low, "partially guaranteed")
	case "garantisiz":
		return strings.Contains(low, "garantisiz") ||
			strings.Contains(low, "non-guaranteed") ||
			strings.Contains(low, "non guaranteed") ||
			strings.Contains(low, "unguaranteed")
	case "istege_bagli":
		return strings.Contains(low, "isteğe bağlı") ||
			strings.Contains(low, "istege bagli") ||
			strings.Contains(low, "on request") ||
			strings.Contains(low, "allotment on request")
	case "serbest_satis":
		return strings.Contains(low, "serbest satış") ||
			strings.Contains(low, "serbest satis") ||
			strings.Contains(low, "free sale")
	case "blok_rezervasyon":
		return strings.Contains(low, "blok rezervasyon") ||
			strings.Contains(low, "block reservation") ||
			strings.Contains(low, "block booking")
	case "blok_satin_alma":
		return strings.Contains(low, "blok satın alma") ||
			strings.Contains(low, "blok satin alma") ||
			strings.Contains(low, "block purchase")
	default:
		return false
	}
}

func sezonMentioned(low, sezon string) bool {
	switch sezon {
	case "yaz":
		return containsPhrase(low, "yaz sezon") ||
			containsPhrase(low, "yazlık") ||
			containsPhrase(low, "yazlik") ||
			containsPhrase(low, "summer season") ||
			containsPhrase(low, "summer period")
	case "kis":
		return containsPhrase(low, "kış sezon") ||
			containsPhrase(low, "kis sezon") ||
			containsPhrase(low, "kışlık") ||
			containsPhrase(low, "kislik") ||
			containsPhrase(low, "winter season") ||
			containsPhrase(low, "winter period")
	case "yillik":
		return containsPhrase(low, "yıllık") ||
			containsPhrase(low, "yillik") ||
			containsPhrase(low, "year-round") ||
			containsPhrase(low, "year round") ||
			containsPhrase(low, "all year")
	default:
		return false
	}
}

func kurEsasiMentioned(low, kur string) bool {
	hasTCMB := strings.Contains(low, "tcmb") ||
		strings.Contains(low, "merkez bankası") ||
		strings.Contains(low, "merkez bankasi") ||
		strings.Contains(low, "central bank")
	switch kur {
	case "giris_gunu_tcmb":
		hasGiris := strings.Contains(low, "giriş") || strings.Contains(low, "giris") ||
			strings.Contains(low, "check-in") || strings.Contains(low, "check in") ||
			strings.Contains(low, "arrival")
		return hasTCMB && hasGiris
	case "cikis_gunu_tcmb":
		hasCikis := strings.Contains(low, "çıkış") || strings.Contains(low, "cikis") ||
			strings.Contains(low, "check-out") || strings.Contains(low, "check out") ||
			strings.Contains(low, "departure")
		return hasTCMB && hasCikis
	case "sabit_kur":
		return strings.Contains(low, "sabit kur") ||
			strings.Contains(low, "fixed rate") ||
			strings.Contains(low, "fixed exchange")
	default:
		return false
	}
}

func containsPhrase(low, phrase string) bool {
	return strings.Contains(low, phrase)
}

func paraBirimiMentioned(lowContract, code string) bool {
	switch code {
	case "EUR":
		return strings.Contains(lowContract, "eur") || strings.Contains(lowContract, "euro") || strings.Contains(lowContract, "€")
	case "GBP":
		return strings.Contains(lowContract, "gbp") || strings.Contains(lowContract, "sterlin") || strings.Contains(lowContract, "pound") || strings.Contains(lowContract, "£")
	case "USD":
		return strings.Contains(lowContract, "usd") || strings.Contains(lowContract, "dollar") || strings.Contains(lowContract, "$")
	case "TRY":
		return containsCurrencyToken(lowContract, "try") ||
			containsCurrencyToken(lowContract, "tl") ||
			strings.Contains(lowContract, "türk lirası") ||
			strings.Contains(lowContract, "turk lirasi") ||
			strings.Contains(lowContract, "₺")
	default:
		return false
	}
}

func containsCurrencyToken(low, tok string) bool {
	for i := 0; i+len(tok) <= len(low); i++ {
		if low[i:i+len(tok)] != tok {
			continue
		}
		leftOK := i == 0 || !isASCIILetterOrDigit(low[i-1])
		rightOK := i+len(tok) == len(low) || !isASCIILetterOrDigit(low[i+len(tok)])
		if leftOK && rightOK {
			return true
		}
	}
	return false
}

func isASCIILetterOrDigit(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

func chunkAEnumString(v any, allowed map[string]struct{}) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	if _, ok := allowed[s]; ok {
		return s, true
	}
	folded := strings.ToLower(strings.ReplaceAll(s, "-", "_"))
	folded = strings.Join(strings.Fields(folded), "_")
	if _, ok := allowed[folded]; ok {
		return folded, true
	}
	return "", false
}

func sanitizeYetkiliMahkeme(v any) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	low := strings.ToLower(s)
	// Uzun cümle: "Antalya mahkemeleri ve icra daireleri yetkilidir" -> Antalya
	if strings.Contains(low, "mahkeme") || strings.Contains(low, "court") || strings.Contains(low, "icra") {
		fields := strings.Fields(s)
		if len(fields) == 0 {
			return "", false
		}
		city := strings.Trim(fields[0], ".,;:")
		if city == "" || len([]rune(city)) > 40 {
			return "", false
		}
		return city, true
	}
	if len([]rune(s)) > 40 {
		return "", false
	}
	return s, true
}

var chunkAOdaTipleri = map[string]struct{}{
	"standart": {}, "suit": {}, "balayi": {}, "engelli": {}, "aile": {}, "deluxe": {},
}

func sanitizeOdaKontenjanlari(v any, contract string) []any {
	arr, ok := v.([]any)
	if !ok {
		arr = []any{}
	}
	filtered := collectOdaByTip(arr, contract, true)
	filtered = enrichKontenjanFromSection(filtered, contract)
	if len(filtered) > 0 {
		return odaMapToList(filtered)
	}
	fallback := collectOdaByTip(arr, contract, false)
	fallback = enrichKontenjanFromSection(fallback, contract)
	return odaMapToList(fallback)
}

func collectOdaByTip(arr []any, contract string, requireText bool) map[string]int {
	byTip := map[string]int{}
	section := kontenjanSection(contract)
	if section == "" {
		section = strings.ToLower(contract)
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tip := chunkAOdaTipi(m["oda_tipi"])
		if tip == "" {
			continue
		}
		adet := intFromJSONNumber(m["adet"])
		if adet <= 0 {
			continue
		}
		if requireText {
			if !odaTipiAndAdetNear(section, tip, m["oda_tipi"], adet) {
				continue
			}
		}
		if prev, exists := byTip[tip]; !exists {
			byTip[tip] = adet
		} else if tip == "engelli" {
			if adet < prev {
				byTip[tip] = adet
			}
		} else if adet > prev {
			byTip[tip] = adet
		}
	}
	return byTip
}

// enrichKontenjanFromSection model atladığında kontenjan maddesinden tip+adet çıkarır.
func enrichKontenjanFromSection(byTip map[string]int, contract string) map[string]int {
	if byTip == nil {
		byTip = map[string]int{}
	}
	section := kontenjanSection(contract)
	if section == "" {
		return byTip
	}
	for _, tip := range []string{"standart", "suit", "balayi", "engelli", "aile", "deluxe"} {
		if _, ok := byTip[tip]; ok {
			continue
		}
		if adet, ok := findAdetNearTip(section, tip); ok {
			byTip[tip] = adet
		}
	}
	return byTip
}

func odaTipiAndAdetNear(section string, tip string, rawTip any, adet int) bool {
	if !odaTipiInKontenjanText(tip, rawTip, section) {
		return false
	}
	return adetNearTip(section, tip, adet)
}

func adetNearTip(section, tip string, adet int) bool {
	syns := odaTipiSynonyms(tip)
	for _, syn := range syns {
		for i := 0; i+len(syn) <= len(section); i++ {
			if section[i:i+len(syn)] != syn {
				continue
			}
			if n, ok := primaryAdetNearSyn(section, i, len(syn)); ok && n == adet {
				return true
			}
		}
	}
	return false
}

func findAdetNearTip(section, tip string) (int, bool) {
	syns := odaTipiSynonyms(tip)
	for _, syn := range syns {
		for i := 0; i+len(syn) <= len(section); i++ {
			if section[i:i+len(syn)] != syn {
				continue
			}
			if n, ok := primaryAdetNearSyn(section, i, len(syn)); ok && n > 0 && n < 10000 {
				return n, true
			}
		}
	}
	return 0, false
}

func primaryAdetNearSyn(section string, synStart, synLen int) (int, bool) {
	// Aynı satırda etiket öncesi sayı: "5 balayı odası"
	beforeStart := synStart - 24
	if beforeStart < 0 {
		beforeStart = 0
	}
	before := section[beforeStart:synStart]
	if idx := strings.LastIndexByte(before, '\n'); idx >= 0 {
		before = before[idx+1:]
	}
	if n, ok := lastIntInText(before); ok {
		return n, true
	}
	// Aynı satırda etiket sonrası sayı: "Standart   240"
	afterEnd := synStart + synLen + 40
	if afterEnd > len(section) {
		afterEnd = len(section)
	}
	after := section[synStart+synLen : afterEnd]
	if idx := strings.IndexByte(after, '\n'); idx >= 0 {
		after = after[:idx]
	}
	return firstIntInChunkText(after)
}

func lastIntInText(s string) (int, bool) {
	last := -1
	end := -1
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			if last < 0 {
				last = i
			}
			end = i + 1
			continue
		}
		if last >= 0 {
			// devam et; en son sayıyı tut
			_ = end
		}
	}
	if last < 0 {
		return 0, false
	}
	// son sayı grubunu bul
	end = len(s)
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] >= '0' && s[i] <= '9' {
			end = i + 1
			j := i
			for j > 0 && s[j-1] >= '0' && s[j-1] <= '9' {
				j--
			}
			n, err := strconv.Atoi(s[j:end])
			return n, err == nil
		}
	}
	return 0, false
}

func containsWholeNumber(s, num string) bool {
	for i := 0; i+len(num) <= len(s); i++ {
		if s[i:i+len(num)] != num {
			continue
		}
		leftOK := i == 0 || !isASCIILetterOrDigit(s[i-1])
		rightOK := i+len(num) == len(s) || !isASCIILetterOrDigit(s[i+len(num)])
		if leftOK && rightOK {
			return true
		}
	}
	return false
}

func odaMapToList(byTip map[string]int) []any {
	if len(byTip) == 0 {
		return []any{}
	}
	order := []string{"standart", "suit", "balayi", "engelli", "aile", "deluxe"}
	out := make([]any, 0, len(byTip))
	for _, tip := range order {
		adet, ok := byTip[tip]
		if !ok {
			continue
		}
		out = append(out, map[string]any{"oda_tipi": tip, "adet": adet})
	}
	return out
}

// chunkAOdaTipi İngilizce/Türkçe oda tipi etiketlerini şema enumuna çevirir.
func chunkAOdaTipi(v any) string {
	s, _ := v.(string)
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.Join(strings.Fields(s), " ")
	switch s {
	case "standart", "standard", "normal", "normal oda", "single", "double", "triple",
		"single room", "double room", "triple room", "standard room":
		return "standart"
	case "suit", "suite", "junior suite", "penthouse", "junior suit":
		return "suit"
	case "balayi", "balayı", "balayı odası", "balayi odasi", "honeymoon", "honeymoon suite", "honeymoon room":
		return "balayi"
	case "engelli", "özürlü", "ozurlu", "özürlü oda", "ozurlu oda", "accessible", "accessible room", "disabled":
		return "engelli"
	case "aile", "aile odası", "aile odasi", "family", "family room":
		return "aile"
	case "deluxe", "deluxe room", "deluxe deniz manzaralı", "deluxe deniz manzarali":
		return "deluxe"
	}
	if _, ok := chunkAOdaTipleri[s]; ok {
		return s
	}
	return ""
}

func odaTipiInKontenjanText(tip string, rawTip any, sectionOrContract string) bool {
	// sectionOrContract: tercihen kontenjan bölümü (fiyat tablosu hariç).
	low := strings.ToLower(sectionOrContract)
	raw, _ := rawTip.(string)
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw != "" && strings.Contains(low, raw) {
		return true
	}
	for _, syn := range odaTipiSynonyms(tip) {
		if strings.Contains(low, syn) {
			return true
		}
	}
	return false
}

func odaTipiSynonyms(tip string) []string {
	switch tip {
	case "standart":
		return []string{"normal oda", "normal room", "standard room", "standart", "standard"}
	case "suit":
		return []string{"junior suite", "suit oda", "penthouse", "suite", "suit"}
	case "balayi":
		return []string{"balayı odası", "balayi odasi", "honeymoon", "balayı", "balayi"}
	case "engelli":
		return []string{"özürlü oda", "ozurlu oda", "accessible", "engelli", "özürlü", "ozurlu"}
	case "aile":
		return []string{"aile odası", "aile odasi", "family room", "family", "aile"}
	case "deluxe":
		return []string{"deluxe deniz manzaralı", "deluxe deniz manzarali", "deluxe deniz", "deluxe room", "deluxe"}
	default:
		return []string{tip}
	}
}

func intFromJSONNumber(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		s := strings.TrimSpace(n)
		if i, err := strconv.Atoi(s); err == nil {
			return i
		}
		if i, ok := firstIntInChunkText(s); ok {
			return i
		}
		return 0
	default:
		return 0
	}
}

func firstIntInChunkText(s string) (int, bool) {
	start := -1
	for i, r := range s {
		if r >= '0' && r <= '9' {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			n, err := strconv.Atoi(s[start:i])
			return n, err == nil
		}
	}
	if start >= 0 {
		n, err := strconv.Atoi(s[start:])
		return n, err == nil
	}
	return 0, false
}

func validAltDonemler(v any) []any {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return []any{}
		}
		ad, _ := m["ad"].(string)
		if ad == "" {
			if alt, ok := m["alt_donem_ad"].(string); ok {
				ad = alt
			}
		}
		if ad == "" {
			return []any{}
		}
		bas, _ := m["baslangic"].(string)
		bit, _ := m["bitis"].(string)
		if bas == "" || bit == "" {
			return []any{}
		}
		out = append(out, map[string]any{
			"ad":        ad,
			"baslangic": bas,
			"bitis":     bit,
		})
	}
	return out
}

func repairChunk(raw string) (map[string]any, error) {
	parsed, err := extract.RepairJSON(raw)
	if err != nil {
		if salvaged := salvageOrphanFields(raw); len(salvaged) > 0 {
			return salvaged, nil
		}
		return nil, err
	}
	// İlk kök nesne erken kapanmış olabilir; dışarıda kalan alanları birleştir.
	if orphans := salvageOrphanFields(raw); len(orphans) > 0 {
		mergeMaps(parsed, orphans)
	}
	return parsed, nil
}

// salvageOrphanFields kök nesne kapandıktan sonra kalan "alan": değer
// parçalarını { ... } içine alıp çözer (Parça A'da sık görülen bozulma).
func salvageOrphanFields(raw string) map[string]any {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	keys := []string{"oda_kontenjanlari", "donem", "meta", "fiyatlar", "release", "stop_sale"}
	var parts []string
	for _, key := range keys {
		needle := `"` + key + `"`
		idx := strings.Index(s, needle)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(s[idx+len(needle):])
		if !strings.HasPrefix(rest, ":") {
			continue
		}
		rest = strings.TrimSpace(rest[1:])
		val, ok := takeJSONValue(rest)
		if !ok || val == "" {
			continue
		}
		parts = append(parts, needle+": "+val)
	}
	if len(parts) == 0 {
		return nil
	}
	wrapped := "{" + strings.Join(parts, ", ") + "}"
	obj, err := extract.RepairJSON(wrapped)
	if err != nil {
		return nil
	}
	return obj
}

func takeJSONValue(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	switch s[0] {
	case '{', '[':
		end, ok := balancedEnd(s)
		if ok {
			return s[:end], true
		}
		// Kesik dizi/nesne: kalan metni al, RepairJSON tamamlar.
		return strings.TrimRight(s, ", \t\r\n"), true
	case '"':
		for i := 1; i < len(s); i++ {
			if s[i] == '\\' {
				i++
				continue
			}
			if s[i] == '"' {
				return s[:i+1], true
			}
		}
		return "", false
	default:
		i := 0
		for i < len(s) && s[i] != ',' && s[i] != '}' && s[i] != ']' && !isSpaceByte(s[i]) {
			i++
		}
		if i == 0 {
			return "", false
		}
		return s[:i], true
	}
}

func balancedEnd(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	open := s[0]
	var close byte
	switch open {
	case '{':
		close = '}'
	case '[':
		close = ']'
	default:
		return 0, false
	}
	depth := 0
	inStr := false
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			continue
		}
		if c == open {
			depth++
		} else if c == close {
			depth--
			if depth == 0 {
				return i + 1, true
			}
		}
	}
	return 0, false
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\n' || b == '\r' || b == '\t'
}

func mergeMaps(dst, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}

func withChunkAgent(ctx context.Context, agent string) context.Context {
	meta, ok := llm.MetaFrom(ctx)
	if !ok {
		return llm.WithMeta(ctx, llm.Meta{Agent: agent})
	}
	meta.Agent = agent
	return llm.WithMeta(ctx, meta)
}
