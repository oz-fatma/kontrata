package extract

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("testdata/%s: %v", name, err)
	}
	return string(b)
}

func TestRepairJSON_MarkdownFence(t *testing.T) {
	raw := readTestdata(t, "markdown_fence.txt")
	got, err := RepairJSON(raw)
	if err != nil {
		t.Fatalf("RepairJSON: %v", err)
	}
	donem, ok := got["donem"].(map[string]any)
	if !ok {
		t.Fatalf("donem yok")
	}
	if donem["baslangic"] != "01.04.2026" {
		t.Fatalf("markdown içinden donem çıkmadı")
	}
}

func TestRepairJSON_MultipleObjects(t *testing.T) {
	raw := readTestdata(t, "multiple_objects.txt")
	got, err := RepairJSON(raw)
	if err != nil {
		t.Fatalf("RepairJSON: %v", err)
	}
	for _, key := range []string{"donem", "oda_kontenjanlari", "fiyatlar", "release", "stop_sale"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("birleşik nesnede %s yok", key)
		}
	}
	rel := got["release"].(map[string]any)
	if rel["gun"] != 10 {
		t.Fatalf("release.gun = %v", rel["gun"])
	}
}

func TestRepairJSON_Truncated(t *testing.T) {
	raw := readTestdata(t, "truncated.txt")
	got, err := RepairJSON(raw)
	if err != nil {
		t.Fatalf("RepairJSON: %v", err)
	}
	if _, ok := got["donem"]; !ok {
		t.Fatalf("kesik JSON'dan donem kurtarılmadı")
	}
	if _, ok := got["oda_kontenjanlari"]; !ok {
		t.Fatalf("kesik JSON'dan oda_kontenjanlari kurtarılmadı")
	}
	if _, ok := got["release"]; ok {
		t.Fatalf("yarım release anahtarı kalmamalı")
	}
}

func TestRepairJSON_Unparseable(t *testing.T) {
	raw := readTestdata(t, "unparseable.txt")
	_, err := RepairJSON(raw)
	if !errors.Is(err, ErrUnparseable) {
		t.Fatalf("err = %v, beklenen ErrUnparseable", err)
	}
}

func TestRepairJSON_ExtraClosers(t *testing.T) {
	raw := readTestdata(t, "extra_closers.txt")
	got, err := RepairJSON(raw)
	if err != nil {
		t.Fatalf("RepairJSON: %v", err)
	}
	donem, ok := got["donem"].(map[string]any)
	if !ok {
		t.Fatalf("donem yok: %v", got)
	}
	if donem["baslangic"] != "2026-04-01" || donem["bitis"] != "2026-10-31" {
		t.Fatalf("donem = %v", donem)
	}
	oda, ok := got["oda_kontenjanlari"].([]any)
	if !ok || len(oda) != 1 {
		t.Fatalf("oda_kontenjanlari = %v", got["oda_kontenjanlari"])
	}
	item := oda[0].(map[string]any)
	if item["adet"] != 170 {
		t.Fatalf("adet = %v", item["adet"])
	}
	rel, ok := got["release"].(map[string]any)
	if !ok {
		t.Fatalf("release yok")
	}
	if rel["gun"] != 10 {
		t.Fatalf("release.gun = %v", rel["gun"])
	}
	ss, ok := got["stop_sale"].([]any)
	if !ok || len(ss) != 0 {
		t.Fatalf("stop_sale = %v", got["stop_sale"])
	}
	if _, ok := got["fiyatlar"]; !ok {
		t.Fatalf("fiyatlar yok")
	}
}

func TestNormalize_TestdataDirtyFields(t *testing.T) {
	raw := readTestdata(t, "dirty_fields.json")
	data, err := RepairJSON(raw)
	if err != nil {
		t.Fatalf("RepairJSON: %v", err)
	}
	got, notes := Normalize(data)
	joined := strings.Join(notes, "\n")
	if _, ok := got["uydurma_alan"]; ok {
		t.Fatalf("uydurma alan kalmış")
	}
	if meta, ok := got["meta"].(map[string]any); ok {
		if _, ok := meta["gizli_not"]; ok {
			t.Fatalf("meta.gizli_not kalmış")
		}
		if meta["sezon"] != "yaz" {
			t.Fatalf("sezon = %v", meta["sezon"])
		}
		if meta["sozlesme_tipi"] != "belirtilmemis" {
			t.Fatalf("bilinmeyen enum belirtilmemis olmalı, %v", meta["sozlesme_tipi"])
		}
	}
	donem := got["donem"].(map[string]any)
	if donem["baslangic"] != "2026-04-01" {
		t.Fatalf("baslangic = %v", donem["baslangic"])
	}
	if donem["bitis"] != "2026-10-31" {
		t.Fatalf("bitis = %v", donem["bitis"])
	}
	if _, ok := donem["stop_sale"]; ok {
		t.Fatalf("stop_sale donem içinde kalmış")
	}
	ss, ok := got["stop_sale"].([]any)
	if !ok || len(ss) != 1 {
		t.Fatalf("taşınan stop_sale yok: %v", got["stop_sale"])
	}
	item := ss[0].(map[string]any)
	if item["baslangic"] != "2026-08-01" {
		t.Fatalf("stop_sale.baslangic = %v", item["baslangic"])
	}
	oda := got["oda_kontenjanlari"].([]any)[0].(map[string]any)
	if oda["adet"] != 1500 {
		t.Fatalf("adet = %v", oda["adet"])
	}
	fiyat := got["fiyatlar"].([]any)[0].(map[string]any)
	if fiyat["tutar"] != 1500 && fiyat["tutar"] != 1500.0 {
		t.Fatalf("tutar = %v", fiyat["tutar"])
	}
	if fiyat["birim"] != "oda_gecelik" {
		t.Fatalf("birim = %v", fiyat["birim"])
	}
	if fiyat["pansiyon"] != "BB" {
		t.Fatalf("pansiyon = %v", fiyat["pansiyon"])
	}
	rel := got["release"].(map[string]any)
	if rel["kapsam"] != "belirtilmemis" {
		t.Fatalf("unknown kapsam = %v", rel["kapsam"])
	}
	if !strings.Contains(joined, "uydurma_alan") {
		t.Fatalf("notlarda uydurma_alan yok: %s", joined)
	}
	if !strings.Contains(joined, "stop_sale donem içinden") {
		t.Fatalf("taşıma notu yok")
	}
}

func TestValidate_AfterNormalize(t *testing.T) {
	raw := readTestdata(t, "markdown_fence.txt")
	data, err := RepairJSON(raw)
	if err != nil {
		t.Fatalf("RepairJSON: %v", err)
	}
	norm, _ := Normalize(data)
	errs := Validate(norm)
	if len(errs) != 0 {
		t.Fatalf("beklenmeyen şema hataları: %v", errs)
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	errs := Validate(map[string]any{"donem": map[string]any{"baslangic": nil, "bitis": nil}})
	if len(errs) == 0 {
		t.Fatalf("eksik zorunlu alanlar için hata beklenirdi")
	}
	for _, e := range errs {
		if strings.Contains(e, "Argos") || strings.Contains(e, "GBP") {
			t.Fatalf("hata sözleşme değeri içeriyor: %s", e)
		}
	}
}

func TestNormalize_FillsRequired(t *testing.T) {
	got, notes := Normalize(map[string]any{"meta": map[string]any{"otel_adi": "X"}})
	for _, key := range []string{"donem", "oda_kontenjanlari", "fiyatlar", "release", "stop_sale"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("zorunlu %s yok", key)
		}
	}
	if len(notes) == 0 {
		t.Fatalf("doldurma notu yok")
	}
}

func TestRepairJSON_DoesNotLogContent(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	secret := "GIZLI-KONTENJAN-999"
	raw := `{"donem": {"baslangic": "2026-04-01", "bitis": "2026-10-31"}, "not": "` + secret + `"}`
	got, err := RepairJSON(raw)
	if err != nil {
		t.Fatalf("RepairJSON: %v", err)
	}
	if got["not"] != secret {
		t.Fatalf("girdi kayboldu")
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("sözleşme içeriği loglandı")
	}
}

func TestEmbeddedSchemaMatchesSource(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "ml", "schema", "kontrat.json"))
	if err != nil {
		t.Fatalf("kaynak şema okunamadı: %v", err)
	}
	if !bytes.Equal(bytes.TrimSpace(src), bytes.TrimSpace(SchemaJSON())) {
		t.Fatalf("gömülü şema ml/schema/kontrat.json ile aynı değil")
	}
}

func TestValidate_DoesNotLogContent(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	secret := "GIZLI-TARIH-2026"
	_ = Validate(map[string]any{
		"donem":             map[string]any{"baslangic": secret, "bitis": nil},
		"oda_kontenjanlari": []any{map[string]any{"oda_tipi": "standart", "adet": 1}},
		"fiyatlar":          []any{map[string]any{"oda_tipi": "standart", "tutar": 1, "birim": "oda_gecelik"}},
		"release":           map[string]any{"gun": 1},
		"stop_sale":         []any{},
	})
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("sözleşme içeriği loglandı")
	}
}

func TestRepairJSON_LogsSnippetWhenDebug(t *testing.T) {
	DebugDump = true
	t.Cleanup(func() { DebugDump = false })

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	marker := "MODEL-CIKTI-KESIT-NESNE-YOK-ABC"
	_, err := RepairJSON(marker + " aciklama, json yok")
	if !errors.Is(err, ErrUnparseable) {
		t.Fatalf("err = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "neden=nesne_yok") {
		t.Fatalf("nesne_yok logu yok: %s", out)
	}
	if !strings.Contains(out, marker) {
		t.Fatalf("kesit loglanmadı: %s", out)
	}
}

func TestRepairJSON_NoSnippetWhenDebugOff(t *testing.T) {
	DebugDump = false
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	marker := "MODEL-CIKTI-KESIT-KAPALI-XYZ"
	_, _ = RepairJSON(marker + " json yok")
	if strings.Contains(buf.String(), marker) {
		t.Fatalf("debug kapalıyken kesit loglandı")
	}
}

func TestValidate_LogsFieldAndConstraint(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	errs := Validate(map[string]any{"donem": map[string]any{"baslangic": nil, "bitis": nil}})
	if len(errs) == 0 {
		t.Fatal("hata bekleniyordu")
	}
	out := buf.String()
	if !strings.Contains(out, "sema hatasi alan=") || !strings.Contains(out, "kisit=") {
		t.Fatalf("alan/kısıt logu yok: %s", out)
	}
	for _, e := range errs {
		if !strings.Contains(e, "alan=") || !strings.Contains(e, "kisit=") {
			t.Fatalf("dönen hata biçimi beklenen değil: %s", e)
		}
	}
}

func TestNormalize_IntegerFromProse(t *testing.T) {
	cases := []struct {
		gun  any
		want int
		note string
	}{
		{"approximately 10 days", 10, "sayı temizlendi: release.gun"},
		{"10 days", 10, "sayı temizlendi: release.gun"},
		{"~10", 10, "sayı temizlendi: release.gun"},
		{"10", 10, "sayı temizlendi: release.gun"},
		{"belirtilmedi", 0, "release.gun metinden çıkarıldı"},
	}
	for _, tc := range cases {
		got, notes := Normalize(map[string]any{
			"release": map[string]any{"gun": tc.gun},
			"oda_kontenjanlari": []any{
				map[string]any{"oda_tipi": "standart", "adet": "approximately 170 rooms"},
			},
			"fiyatlar": []any{
				map[string]any{"oda_tipi": "standart", "tutar": "approx 50 EUR", "birim": "oda_gecelik"},
			},
		})
		rel := got["release"].(map[string]any)
		if rel["gun"] != tc.want {
			t.Fatalf("gun %q = %v, beklenen %d", tc.gun, rel["gun"], tc.want)
		}
		joined := strings.Join(notes, "\n")
		if !strings.Contains(joined, tc.note) {
			t.Fatalf("gun %q notu yok: %s", tc.gun, joined)
		}
		oda := got["oda_kontenjanlari"].([]any)[0].(map[string]any)
		if oda["adet"] != 170 {
			t.Fatalf("adet = %v", oda["adet"])
		}
		fiyat := got["fiyatlar"].([]any)[0].(map[string]any)
		if fiyat["tutar"] != 50 && fiyat["tutar"] != 50.0 {
			t.Fatalf("tutar = %v", fiyat["tutar"])
		}
		if errs := Validate(got); len(errs) != 0 {
			t.Fatalf("gun %q şema hataları: %v", tc.gun, errs)
		}
	}
}
