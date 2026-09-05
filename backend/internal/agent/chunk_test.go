package agent

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/oz-fatma/kontrata/backend/internal/llm"
)

const (
	chunkJSONA = `{"donem": {"baslangic": "2026-04-01", "bitis": "2026-10-31", "alt_donemler": []}, "oda_kontenjanlari": [{"oda_tipi": "standart", "adet": 10}]}`
	chunkJSONB = `{"fiyatlar": [{"oda_tipi": "standart", "tutar": 85, "birim": "oda_gecelik", "pansiyon": "HB"}]}`
	chunkJSONC = `{"release": {"gun": 21, "kapsam": "kontenjan_iadesi"}, "stop_sale": []}`
	chunkJSOND = `{"meta": {"otel_adi": "Argos Otel", "acente_adi": "Side Turizm", "para_birimi": "GBP"}}`
)

type agentStubLLM struct {
	mu     sync.Mutex
	queues map[string][]string
	idx    map[string]int
	agents []string
}

func newAgentStub(queues map[string][]string) *agentStubLLM {
	return &agentStubLLM{queues: queues, idx: make(map[string]int)}
}

func (s *agentStubLLM) Generate(ctx context.Context, _, _ string) (string, error) {
	meta, ok := llm.MetaFrom(ctx)
	agent := llm.AgentReader
	if ok && meta.Agent != "" {
		agent = meta.Agent
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents = append(s.agents, agent)
	q := s.queues[agent]
	i := s.idx[agent]
	if i >= len(q) {
		return "", llm.ErrUnavailable
	}
	out := q[i]
	s.idx[agent] = i + 1
	return out, nil
}

func (s *agentStubLLM) callCount(agent string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.idx[agent]
}

func TestReader_ChunkedMerge(t *testing.T) {
	t.Setenv("EXTRACT_MODE", ExtractModeChunked)

	stub := newAgentStub(map[string][]string{
		llm.AgentReaderChunkA: {chunkJSONA},
		llm.AgentReaderChunkB: {chunkJSONB},
		llm.AgentReaderChunkC: {chunkJSONC},
		llm.AgentReaderChunkD: {chunkJSOND},
	})
	r := &Reader{LLM: stub}
	res, err := r.Extract(context.Background(), []string{"Argos Otel / Side Turizm. Sezon 1 Nisan 2026 - 31 Ekim 2026. Standart oda 10 adet, 85 GBP."})
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if len(res.SchemaErrors) != 0 {
		t.Fatalf("şema hataları: %v", res.SchemaErrors)
	}
	if stub.callCount(llm.AgentReaderChunkA) != 1 || stub.callCount(llm.AgentReaderChunkB) != 1 || stub.callCount(llm.AgentReaderChunkC) != 1 || stub.callCount(llm.AgentReaderChunkD) != 1 {
		t.Fatalf("parça çağrıları A=%d B=%d C=%d D=%d", stub.callCount(llm.AgentReaderChunkA), stub.callCount(llm.AgentReaderChunkB), stub.callCount(llm.AgentReaderChunkC), stub.callCount(llm.AgentReaderChunkD))
	}
	donem, _ := res.Data["donem"].(map[string]any)
	if donem["baslangic"] != "2026-04-01" {
		t.Fatalf("donem.baslangic = %v", donem["baslangic"])
	}
	fiyatlar, ok := res.Data["fiyatlar"].([]any)
	if !ok || len(fiyatlar) != 1 {
		t.Fatalf("fiyatlar = %v", res.Data["fiyatlar"])
	}
	release, _ := res.Data["release"].(map[string]any)
	if release["gun"] != float64(21) && release["gun"] != int(21) {
		t.Fatalf("release.gun = %v", release["gun"])
	}
	meta, _ := res.Data["meta"].(map[string]any)
	if meta["para_birimi"] != "GBP" {
		t.Fatalf("meta = %v", meta)
	}
	wantAgents := []string{llm.AgentReaderChunkA, llm.AgentReaderChunkB, llm.AgentReaderChunkC, llm.AgentReaderChunkD}
	got := slices.Clone(stub.agents)
	slices.Sort(got)
	slices.Sort(wantAgents)
	if !slices.Equal(got, wantAgents) {
		t.Fatalf("agent etiketleri = %v", stub.agents)
	}
}

func TestReader_ChunkedPartialFailure(t *testing.T) {
	t.Setenv("EXTRACT_MODE", ExtractModeChunked)

	stub := newAgentStub(map[string][]string{
		llm.AgentReaderChunkA: {chunkJSONA},
		llm.AgentReaderChunkB: {"bu json değil", "hala değil"},
		llm.AgentReaderChunkC: {chunkJSONC},
		llm.AgentReaderChunkD: {chunkJSOND},
	})
	r := &Reader{LLM: stub}
	res, err := r.Extract(context.Background(), []string{"Standart oda kontenjanı 10 adet."})
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if stub.callCount(llm.AgentReaderChunkA) != 1 {
		t.Fatalf("parça A etkilendi: çağrı=%d", stub.callCount(llm.AgentReaderChunkA))
	}
	if stub.callCount(llm.AgentReaderChunkC) != 1 {
		t.Fatalf("parça C etkilendi: çağrı=%d", stub.callCount(llm.AgentReaderChunkC))
	}
	if stub.callCount(llm.AgentReaderChunkB) != 2 {
		t.Fatalf("parça B retry bekleniyordu: çağrı=%d", stub.callCount(llm.AgentReaderChunkB))
	}
	if res.RetryCount != 1 {
		t.Fatalf("RetryCount = %d", res.RetryCount)
	}
	donem, _ := res.Data["donem"].(map[string]any)
	if donem["baslangic"] != "2026-04-01" {
		t.Fatalf("birleşimde donem kayboldu: %v", donem)
	}
	fiyatlar, _ := res.Data["fiyatlar"].([]any)
	if len(fiyatlar) != 0 {
		t.Fatalf("başarısız parça B fiyat üretmemeli, fiyatlar=%v", fiyatlar)
	}
	release, _ := res.Data["release"].(map[string]any)
	if release == nil {
		t.Fatal("parça C release birleşmedi")
	}
	if len(res.SchemaErrors) == 0 {
		t.Fatal("eksik fiyatlar için şema hatası bekleniyordu")
	}
}

func TestReader_SingleModeRegression(t *testing.T) {
	t.Setenv("EXTRACT_MODE", ExtractModeSingle)

	stub := &stubLLM{responses: []string{validJSON}}
	r := &Reader{LLM: stub}
	res, err := r.Extract(context.Background(), []string{"standart oda kontenjanı 10"})
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if stub.calls != 1 {
		t.Fatalf("single modda tek çağrı bekleniyordu, çağrı=%d", stub.calls)
	}
	if len(res.SchemaErrors) != 0 {
		t.Fatalf("şema hataları: %v", res.SchemaErrors)
	}
}

func TestExtractModeFromEnv(t *testing.T) {
	t.Setenv("EXTRACT_MODE", "")
	if got := ExtractModeFromEnv(); got != ExtractModeSingle {
		t.Fatalf("varsayılan = %q", got)
	}
	t.Setenv("EXTRACT_MODE", "chunked")
	if got := ExtractModeFromEnv(); got != ExtractModeChunked {
		t.Fatalf("chunked = %q", got)
	}
	t.Setenv("EXTRACT_MODE", "CHUNKED")
	if got := ExtractModeFromEnv(); got != ExtractModeChunked {
		t.Fatalf("büyük harf chunked = %q", got)
	}
}

func TestMergeMaps(t *testing.T) {
	dst := map[string]any{"donem": map[string]any{"baslangic": "2026-04-01"}}
	src := map[string]any{"fiyatlar": []any{map[string]any{"tutar": 85}}}
	mergeMaps(dst, src)
	if _, ok := dst["donem"]; !ok {
		t.Fatal("donem kayboldu")
	}
	if _, ok := dst["fiyatlar"]; !ok {
		t.Fatal("fiyatlar eklenmedi")
	}
}

func TestSanitizeChunkA(t *testing.T) {
	raw := map[string]any{
		"donem":        map[string]any{"baslangic": "2026-04-01", "bitis": "2026-10-31"},
		"alt_donemler": []any{map[string]any{"foo": "bar"}},
		"meta": map[string]any{
			"otel_adi": "Argos Otel",
		},
		"kontenjan_detaları": "uydurma",
		"oda_kontenjanlari": []any{
			map[string]any{"oda_tipi": "standart", "adet": 170},
		},
	}
	got := sanitizeChunkA(raw, "170 normal oda, 20 suit oda, 5 balayı odası ve 1 özürlü odası")
	if _, ok := got["meta"]; ok {
		t.Fatal("parça A meta içermemeli")
	}
	if _, ok := got["kontenjan_detaları"]; ok {
		t.Fatal("uydurma alan kaldirilmadi")
	}
	if _, ok := got["alt_donemler"]; ok {
		t.Fatal("ust seviye alt_donemler kalmamali")
	}
	donem, _ := got["donem"].(map[string]any)
	alt, _ := donem["alt_donemler"].([]any)
	if len(alt) != 0 {
		t.Fatalf("gecersiz alt_donemler bos olmali, got=%v", alt)
	}
}

func TestSalvageOrphanOdaKontenjanlari(t *testing.T) {
	raw := `{"donem": {"baslangic": "2026-04-01", "bitis": "2026-10-31"}, "alt_donemler": [{"oda_tipi": "standart", "alt_donem_ad": "x"}}]}, "oda_kontenjanlari": [{"oda_tipi": "standart", "adet": 170}, {"oda_tipi": "suit", "adet": 20}, {"oda_tipi": "balayi", "adet": 5}, {"oda_tipi": "engelli", "adet": 1}`
	got, err := repairChunk(raw)
	if err != nil {
		t.Fatalf("repairChunk: %v", err)
	}
	san := sanitizeChunkA(got, "170 normal oda, 20 suit oda, 5 balayı odası ve 1 özürlü odası")
	oda, _ := san["oda_kontenjanlari"].([]any)
	if len(oda) != 4 {
		t.Fatalf("oda=%v keys=%v", oda, got)
	}
}

func TestSanitizeChunkD(t *testing.T) {
	raw := map[string]any{
		"meta": map[string]any{
			"otel_adi":        "Argos Otel",
			"acente_adi":      "Side Turizm",
			"para_birimi":     "gbp",
			"kur_esasi":       "giris_gunu_tcmb",
			"yetkili_mahkeme": "Antalya mahkemeleri ve icra daireleri yetkilidir",
			"sozlesme_tipi":   "YETKİLİ_SÖZLEŞME",
			"sezon":           "yellivis",
			"uydurma_alan":    "x",
		},
		"donem": map[string]any{"baslangic": "2026-04-01"},
	}
	got := sanitizeChunkD(raw, "Side Turizm ile Argos Otel, Antalya mahkemeleri, faturalar giriş günündeki Merkez Bankası kuru ile İngiliz Sterlini (GBP)")
	if _, ok := got["donem"]; ok {
		t.Fatal("parça D donem içermemeli")
	}
	meta, ok := got["meta"].(map[string]any)
	if !ok {
		t.Fatal("meta yok")
	}
	if meta["para_birimi"] != "GBP" {
		t.Fatalf("para_birimi = %v", meta["para_birimi"])
	}
	if meta["yetkili_mahkeme"] != "Antalya" {
		t.Fatalf("yetkili_mahkeme = %v", meta["yetkili_mahkeme"])
	}
	if meta["kur_esasi"] != "giris_gunu_tcmb" {
		t.Fatalf("kur_esasi = %v", meta["kur_esasi"])
	}
	if _, ok := meta["sozlesme_tipi"]; ok {
		t.Fatalf("gecersiz sozlesme_tipi silinmeli: %v", meta["sozlesme_tipi"])
	}
	if _, ok := meta["sezon"]; ok {
		t.Fatalf("gecersiz sezon silinmeli: %v", meta["sezon"])
	}
	if _, ok := meta["uydurma_alan"]; ok {
		t.Fatal("meta uydurma alt alan kalmamali")
	}
}

func TestSanitizeChunkD_DropsGuessedEnumsAndBackfills(t *testing.T) {
	contract := `Taraflar: TUI Turizm A.Ş. (Operatör) — Kemer Resort Hotel (Tesis)
Erken sezon / Yüksek sezon / Geç sezon fiyatları EUR cinsindendir.
operatör hesabına yazılır.`
	raw := map[string]any{
		"meta": map[string]any{
			"otel_adi":      "Uydurma Palace",
			"sozlesme_tipi": "kismen_garantili",
			"sezon":         "yaz",
			"kur_esasi":     "sabit_kur",
			"para_birimi":   "TRY",
		},
	}
	got := sanitizeChunkD(raw, contract)
	meta, ok := got["meta"].(map[string]any)
	if !ok {
		t.Fatal("meta yok")
	}
	if _, ok := meta["sozlesme_tipi"]; ok {
		t.Fatalf("uydurma sozlesme_tipi kalmamali: %v", meta["sozlesme_tipi"])
	}
	if _, ok := meta["sezon"]; ok {
		t.Fatalf("erken/yüksek sezon adından yaz uydurulmamali: %v", meta["sezon"])
	}
	if _, ok := meta["kur_esasi"]; ok {
		t.Fatalf("kanıtsız kur_esasi kalmamali: %v", meta["kur_esasi"])
	}
	if meta["para_birimi"] != "EUR" {
		t.Fatalf("para_birimi backfill EUR olmali, got %v", meta["para_birimi"])
	}
	otel, _ := meta["otel_adi"].(string)
	if !strings.Contains(strings.ToLower(otel), "kemer") {
		t.Fatalf("otel_adi backfill: %q", otel)
	}
	acente, _ := meta["acente_adi"].(string)
	if !strings.Contains(strings.ToLower(acente), "tui") {
		t.Fatalf("acente_adi backfill: %q", acente)
	}
}

func TestSanitizeChunkD_KeepsEvidenceBasedEnums(t *testing.T) {
	contract := "Tamamen garantili kontenjan. Yaz sezonu 2026. Faturalar sabit kur ile EUR."
	raw := map[string]any{
		"meta": map[string]any{
			"sozlesme_tipi": "tamamen_garantili",
			"sezon":         "yaz",
			"kur_esasi":     "sabit_kur",
			"para_birimi":   "EUR",
		},
	}
	got := sanitizeChunkD(raw, contract)
	meta := got["meta"].(map[string]any)
	if meta["sozlesme_tipi"] != "tamamen_garantili" {
		t.Fatalf("sozlesme_tipi = %v", meta["sozlesme_tipi"])
	}
	if meta["sezon"] != "yaz" {
		t.Fatalf("sezon = %v", meta["sezon"])
	}
	if meta["kur_esasi"] != "sabit_kur" {
		t.Fatalf("kur_esasi = %v", meta["kur_esasi"])
	}
}

func TestSanitizeChunkAOdaDedup(t *testing.T) {
	raw := []any{
		map[string]any{"oda_tipi": "standart", "adet": 170},
		map[string]any{"oda_tipi": "suit", "adet": 20},
		map[string]any{"oda_tipi": "balayi", "adet": 5},
		map[string]any{"oda_tipi": "engelli", "adet": 1},
		map[string]any{"oda_tipi": "özürlü", "adet": 80},
	}
	got := sanitizeOdaKontenjanlari(raw, "170 normal oda, 20 suit oda, 5 balayı odası ve 1 özürlü odası")
	if len(got) != 4 {
		t.Fatalf("adet=%d", len(got))
	}
	last := got[3].(map[string]any)
	if last["oda_tipi"] != "engelli" || intFromJSONNumber(last["adet"]) != 1 {
		t.Fatalf("engelli = %v", last)
	}
	standart := got[0].(map[string]any)
	if intFromJSONNumber(standart["adet"]) != 170 {
		t.Fatalf("standart kontenjan buyuk adet korunmali = %v", standart)
	}
}

func TestSanitizeChunkA_EnglishRoomTypes(t *testing.T) {
	contract := "Allocation: 150 standard rooms, 40 family rooms, 10 junior suite, 5 honeymoon, 2 accessible, 8 deluxe."
	raw := map[string]any{
		"donem": map[string]any{"baslangic": "2026-05-01", "bitis": "2026-10-31", "alt_donemler": []any{}},
		"oda_kontenjanlari": []any{
			map[string]any{"oda_tipi": "standard", "adet": 150},
			map[string]any{"oda_tipi": "family", "adet": 40},
			map[string]any{"oda_tipi": "junior suite", "adet": 10},
			map[string]any{"oda_tipi": "honeymoon", "adet": 5},
			map[string]any{"oda_tipi": "accessible", "adet": 2},
			map[string]any{"oda_tipi": "deluxe", "adet": 8},
			map[string]any{"oda_tipi": "villa", "adet": 99}, // eşlenmeyen tip, elenmeli
		},
	}
	got := sanitizeChunkA(raw, contract)
	oda := got["oda_kontenjanlari"].([]any)
	want := map[string]int{
		"standart": 150,
		"aile":     40,
		"suit":     10,
		"balayi":   5,
		"engelli":  2,
		"deluxe":   8,
	}
	if len(oda) != len(want) {
		t.Fatalf("adet=%d oda=%v", len(oda), oda)
	}
	for _, item := range oda {
		m := item.(map[string]any)
		tip := m["oda_tipi"].(string)
		if intFromJSONNumber(m["adet"]) != want[tip] {
			t.Fatalf("%s = %v", tip, m)
		}
	}
}

func TestSanitizeChunkB_EnglishRoomTypes(t *testing.T) {
	got := sanitizeChunkB(map[string]any{
		"fiyatlar": []any{
			map[string]any{"oda_tipi": "standard", "tutar": 65},
			map[string]any{"oda_tipi": "family", "tutar": 98},
		},
	}, "")
	arr := got["fiyatlar"].([]any)
	if arr[0].(map[string]any)["oda_tipi"] != "standart" {
		t.Fatalf("standard -> %v", arr[0])
	}
	if arr[1].(map[string]any)["oda_tipi"] != "aile" {
		t.Fatalf("family -> %v", arr[1])
	}
}

func TestSanitizeChunkB_FixesBadBirimAndNullAltDonem(t *testing.T) {
	contract := "Aşağıdaki fiyatlar kişi başı gecelik, her şey dahil (AI) pansiyon esasına göre EUR cinsindendir."
	raw := map[string]any{
		"fiyatlar": []any{
			map[string]any{"oda_tipi": "standart", "tutar": 48.0, "birim": "odeme_kontenjan_emi", "pansiyon": "RO", "alt_donem_ad": nil},
			map[string]any{"oda_tipi": "standart", "tutar": 48.0, "birim": "odeme_kontenjan_emi", "pansiyon": "RO", "alt_donem_ad": nil},
			map[string]any{"oda_tipi": "aile", "tutar": 61.0, "birim": "x", "alt_donem_ad": "Erken sezon"},
		},
	}
	got := sanitizeChunkB(raw, contract)
	arr := got["fiyatlar"].([]any)
	if len(arr) != 2 {
		t.Fatalf("dedupe sonrası adet=%d: %v", len(arr), arr)
	}
	first := arr[0].(map[string]any)
	if first["birim"] != "kisi_gecelik" {
		t.Fatalf("birim = %v", first["birim"])
	}
	if first["pansiyon"] != "AI" {
		t.Fatalf("pansiyon = %v (AI beklenir)", first["pansiyon"])
	}
	if _, ok := first["alt_donem_ad"]; ok {
		t.Fatalf("null alt_donem_ad kalmamali: %v", first)
	}
}

func TestSanitizeChunkA_PreservesInvertedDates(t *testing.T) {
	raw := map[string]any{
		"donem": map[string]any{"baslangic": "2026-05-01", "bitis": "2026-04-20", "alt_donemler": []any{}},
		"oda_kontenjanlari": []any{
			map[string]any{"oda_tipi": "standart", "adet": 150},
			map[string]any{"oda_tipi": "aile", "adet": 40},
		},
	}
	got := sanitizeChunkA(raw, "Standard 150 Family 40")
	d := got["donem"].(map[string]any)
	if d["baslangic"] != "2026-05-01" || d["bitis"] != "2026-04-20" {
		t.Fatalf("ters tarih korunmali (swap yok): %v", d)
	}
}

func TestSanitizeChunkA_DropsPriceBleedIntoKontenjan(t *testing.T) {
	contract := `ARTICLE 2 — ROOM ALLOTMENT
Room type                        Quantity
Standard                        150
Family                        40
ARTICLE 3 — NAME LIST
ARTICLE 4 — RATES
Standard                        65,00
Family                        98,00
Junior suite                        140,00
Penthouse                        260,00`
	raw := map[string]any{
		"donem": map[string]any{"baslangic": "2026-05-01", "bitis": "2026-04-20"},
		"oda_kontenjanlari": []any{
			map[string]any{"oda_tipi": "standard", "adet": 150},
			map[string]any{"oda_tipi": "family", "adet": 40},
			map[string]any{"oda_tipi": "junior_suite", "adet": 140},
			map[string]any{"oda_tipi": "penthouse", "adet": 260},
			map[string]any{"oda_tipi": "suit", "adet": 260},
		},
	}
	got := sanitizeChunkA(raw, contract)
	oda := got["oda_kontenjanlari"].([]any)
	if len(oda) != 2 {
		t.Fatalf("adet=%d want 2: %v", len(oda), oda)
	}
	tips := map[string]int{}
	for _, item := range oda {
		m := item.(map[string]any)
		tips[m["oda_tipi"].(string)] = intFromJSONNumber(m["adet"])
	}
	if tips["standart"] != 150 || tips["aile"] != 40 {
		t.Fatalf("kontenjan = %v", tips)
	}
	if _, ok := tips["suit"]; ok {
		t.Fatal("fiyat satırından suit/penthouse kontenjana girmemeli")
	}
}

func TestSanitizeChunkC_StopSale(t *testing.T) {
	contract := `5. SATIŞ DURDURMA (STOP-SALE)
10.07.2026               18.07.2026               Tüm oda tipleri
20.08.2026               24.08.2026               Suit`
	raw := map[string]any{
		"release": []any{map[string]any{"gun": 14, "kapsam": "isim_listesi"}},
		"stop_sale": []any{
			[]any{"standart", "Erken sezon"},
			map[string]any{"baslangic": "2026-07-10", "bitis": "2026-07-18", "kapsam": "tum_odai_tipleri"},
			map[string]any{"baslangic": "2026-08-20", "bitis": "2026-08-24", "kapsam": "Suit"},
			map[string]any{"kapsam": "her_ikisi"}, // tarihsiz — düş
		},
	}
	got := sanitizeChunkC(raw, contract)
	rel, ok := got["release"].(map[string]any)
	if !ok || intFromJSONNumber(rel["gun"]) != 14 {
		t.Fatalf("release = %v", got["release"])
	}
	ss := got["stop_sale"].([]any)
	if len(ss) != 2 {
		t.Fatalf("stop_sale = %v", ss)
	}
	first := ss[0].(map[string]any)
	if first["kapsam"] != "tüm oda tipleri" {
		t.Fatalf("kapsam normalize = %v", first["kapsam"])
	}
}

func TestSanitizeChunkC_DropsInventedStopSale(t *testing.T) {
	// Argos: stop-sale maddesi yok; model dönem tarihlerini stop_sale yazmış.
	contract := `MADDE 3 — İSİM LİSTESİ
Acente 10 gün önce bildirir.
MADDE 8 — ODA FİYATLARI`
	raw := map[string]any{
		"release":   map[string]any{"gun": 10, "kapsam": "isim_listesi"},
		"stop_sale": []any{map[string]any{"baslangic": "2026-04-01", "bitis": "2026-10-31", "kapsam": "her_ikisi"}},
	}
	got := sanitizeChunkC(raw, contract)
	if len(got["stop_sale"].([]any)) != 0 {
		t.Fatalf("uydurma stop_sale kalmamali: %v", got["stop_sale"])
	}
}

func TestEnrichKontenjanFromSection_ArgosAndTUI(t *testing.T) {
	argos := `MADDE 2 — ODA KONTENJANI
Otel, acenteye 170 normal oda, 20 suit oda, 5 balayı odası ve 1 özürlü odası tahsis etmeyi kabul eder.
MADDE 3 — İSİM LİSTESİ`
	got := sanitizeOdaKontenjanlari([]any{
		map[string]any{"oda_tipi": "standart", "adet": 170},
		map[string]any{"oda_tipi": "suit", "adet": 20},
		map[string]any{"oda_tipi": "balayi", "adet": 1}, // yanlış adet — yakınlık eler, enrich 5 yazar
	}, argos)
	tips := map[string]int{}
	for _, item := range got {
		m := item.(map[string]any)
		tips[m["oda_tipi"].(string)] = intFromJSONNumber(m["adet"])
	}
	if tips["balayi"] != 5 {
		t.Fatalf("balayi adet=%v want 5 (enrich)", tips)
	}
	if tips["engelli"] != 1 {
		t.Fatalf("engelli eksik: %v", tips)
	}

	tui := `2. ODA KONTENJANI
Oda tipi                        Adet
Standart                        240
Aile odası                        45
Deluxe deniz manzaralı                     30
Suit                        12
3. RELEASE SÜRESİ`
	got2 := sanitizeOdaKontenjanlari([]any{
		map[string]any{"oda_tipi": "standart", "adet": 240},
		map[string]any{"oda_tipi": "suit", "adet": 12},
	}, tui)
	tips2 := map[string]int{}
	for _, item := range got2 {
		m := item.(map[string]any)
		tips2[m["oda_tipi"].(string)] = intFromJSONNumber(m["adet"])
	}
	if tips2["aile"] != 45 || tips2["deluxe"] != 30 {
		t.Fatalf("tui enrich = %v", tips2)
	}
}

func TestEnrichFiyatlarFromContract_Coral(t *testing.T) {
	contract := `ARTICLE 4 — RATES
Rates are per room per night in EUR, bed and breakfast basis.
Standard                        65,00
Family                        98,00
Junior suite                        140,00
Penthouse                        260,00
ARTICLE 5 — PAYMENT`
	raw := map[string]any{
		"fiyatlar": []any{
			map[string]any{"oda_tipi": "standart", "tutar": 65.0, "birim": "oda_gecelik", "pansiyon": "BB"},
			map[string]any{"oda_tipi": "aile", "tutar": 98.0, "birim": "oda_gecelik", "pansiyon": "BB"},
		},
	}
	got := sanitizeChunkB(raw, contract)
	arr := got["fiyatlar"].([]any)
	tutars := map[float64]string{}
	for _, item := range arr {
		m := item.(map[string]any)
		tutar, _ := m["tutar"].(float64)
		tutars[tutar] = m["oda_tipi"].(string)
	}
	if tutars[140] != "suit" || tutars[260] != "suit" {
		t.Fatalf("junior/penthouse enrich yok: %v", arr)
	}
}

func TestSanitizeChunkD_BackfillsKurEsasi(t *testing.T) {
	contract := `Side Turizm ile Argos Otel arasında.
Faturalar, müşterinin otele giriş günündeki Türkiye Cumhuriyet Merkez Bankası kuru esas alınarak İngiliz Sterlini (GBP) üzerinden düzenlenir.
Antalya mahkemeleri`
	raw := map[string]any{
		"meta": map[string]any{
			"otel_adi":    "Argos Otel",
			"acente_adi":  "Side Turizm",
			"para_birimi": "GBP",
		},
	}
	got := sanitizeChunkD(raw, contract)
	meta := got["meta"].(map[string]any)
	if meta["kur_esasi"] != "giris_gunu_tcmb" {
		t.Fatalf("kur_esasi = %v", meta["kur_esasi"])
	}
}

func TestFiyatlarTooFew(t *testing.T) {
	contract := `ARTICLE 4 — RATES
Standard                        65,00
Family                        98,00
Junior suite                        140,00
Penthouse                        260,00
ARTICLE 5 — PAYMENT`
	one := []any{map[string]any{"oda_tipi": "standart", "tutar": 65.0}}
	if !fiyatlarTooFew(one, contract) {
		t.Fatal("tek satır tooFew olmali")
	}
	four := []any{
		map[string]any{"oda_tipi": "standart", "tutar": 65.0},
		map[string]any{"oda_tipi": "aile", "tutar": 98.0},
		map[string]any{"oda_tipi": "suit", "tutar": 140.0},
		map[string]any{"oda_tipi": "suit", "tutar": 260.0},
	}
	if fiyatlarTooFew(four, contract) {
		t.Fatal("4 satır yeterli")
	}
}

func TestChunkAOdaTipiMapping(t *testing.T) {
	cases := map[string]string{
		"standard":     "standart",
		"family":       "aile",
		"suite":        "suit",
		"junior suite": "suit",
		"penthouse":    "suit",
		"honeymoon":    "balayi",
		"accessible":   "engelli",
		"deluxe":       "deluxe",
		"single":       "standart",
		"double":       "standart",
		"triple":       "standart",
	}
	for in, want := range cases {
		if got := chunkAOdaTipi(in); got != want {
			t.Fatalf("%q -> %q, beklenen %q", in, got, want)
		}
	}
}

func TestChunkB_UnusableTriggersRetry(t *testing.T) {
	t.Setenv("EXTRACT_MODE", ExtractModeChunked)
	stub := newAgentStub(map[string][]string{
		llm.AgentReaderChunkA: {chunkJSONA},
		llm.AgentReaderChunkB: {
			`{"code": "print('hello')", "note": "not prices"}`,
			chunkJSONB,
		},
		llm.AgentReaderChunkC: {chunkJSONC},
		llm.AgentReaderChunkD: {chunkJSOND},
	})
	r := &Reader{LLM: stub}
	res, err := r.Extract(context.Background(), []string{"Standart oda 10 adet, fiyat 85 EUR."})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if stub.callCount(llm.AgentReaderChunkB) != 2 {
		t.Fatalf("parça B retry bekleniyordu: çağrı=%d", stub.callCount(llm.AgentReaderChunkB))
	}
	fiyatlar, _ := res.Data["fiyatlar"].([]any)
	if len(fiyatlar) == 0 {
		t.Fatal("retry sonrası fiyatlar boş")
	}
}
