package agent

import (
	"testing"
)

func TestConfidence_ModelUntouched(t *testing.T) {
	got := confidenceFor("donem", []string{"bilinmeyen alan atıldı: uydurma_alan"})
	if got != confidenceModel {
		t.Fatalf("güven = %v, beklenen %v", got, confidenceModel)
	}
}

func TestConfidence_Normalized(t *testing.T) {
	got := confidenceFor("donem", []string{"tarih ISO-8601 yapıldı: donem.baslangic"})
	if got != confidenceRepaired {
		t.Fatalf("güven = %v, beklenen %v", got, confidenceRepaired)
	}
}

func TestConfidence_FilledDefault(t *testing.T) {
	got := confidenceFor("release", []string{"zorunlu alan dolduruldu: release"})
	if got != confidenceFilled {
		t.Fatalf("güven = %v, beklenen %v", got, confidenceFilled)
	}
}

func TestConfidence_FilledWinsOverRepair(t *testing.T) {
	got := confidenceFor("donem", []string{
		"zorunlu alan dolduruldu: donem.baslangic",
		"tarih ISO-8601 yapıldı: donem.bitis",
	})
	if got != confidenceFilled {
		t.Fatalf("güven = %v, beklenen %v", got, confidenceFilled)
	}
}

func TestSourcePage_FindsFirstPage(t *testing.T) {
	pages := []string{
		"Giriş metni, otel tanıtımı.",
		"Dönem: 2026-04-01 ile 2026-10-31 arası geçerlidir.",
		"Tekrar 2026-04-01.",
	}
	p := sourcePage(map[string]any{"baslangic": "2026-04-01", "bitis": "2026-10-31"}, pages)
	if p == nil || *p != 2 {
		t.Fatalf("sayfa = %v, beklenen 2", p)
	}
}

func TestSourcePage_MissingIsNil(t *testing.T) {
	p := sourcePage(map[string]any{"baslangic": "2099-01-01"}, []string{"başka bir metin"})
	if p != nil {
		t.Fatalf("sayfa = %v, beklenen nil", *p)
	}
}

func TestBuildExtractionMeta_GraphQLPaths(t *testing.T) {
	data := map[string]any{
		"donem":             map[string]any{"baslangic": "2026-04-01", "bitis": "2026-10-31"},
		"oda_kontenjanlari": []any{map[string]any{"oda_tipi": "standart", "adet": 10}},
		"fiyatlar":          []any{map[string]any{"oda_tipi": "standart", "tutar": 85.0, "birim": "oda_gecelik"}},
		"release":           map[string]any{"gun": 21, "kapsam": "kontenjan_iadesi"},
		"stop_sale":         []any{},
	}
	pages := []string{"standart oda 10 adet, 2026-04-01 — 2026-10-31, 85 EUR, release 21 gün"}
	meta := BuildExtractionMeta(data, nil, pages)
	if len(meta) != 5 {
		t.Fatalf("alan sayısı = %d", len(meta))
	}
	byPath := map[string]FieldMeta{}
	for _, m := range meta {
		byPath[m.FieldPath] = m
	}
	for _, path := range []string{"donem", "odaKontenjanlari", "fiyatlar", "release", "stopSale"} {
		if _, ok := byPath[path]; !ok {
			t.Fatalf("eksik alan yolu: %s", path)
		}
	}
	if byPath["donem"].SourcePage == nil || *byPath["donem"].SourcePage != 1 {
		t.Fatalf("donem kaynak sayfa = %v", byPath["donem"].SourcePage)
	}
	if byPath["stopSale"].SourcePage != nil {
		t.Fatalf("boş stop_sale için sayfa beklenmez")
	}
}
