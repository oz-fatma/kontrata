package agent

import (
	"strings"
	"testing"
)

func validContract() map[string]any {
	return map[string]any{
		"donem": map[string]any{
			"baslangic":    "2026-04-01",
			"bitis":        "2026-10-31",
			"alt_donemler": []any{},
		},
		"oda_kontenjanlari": []any{
			map[string]any{"oda_tipi": "standart", "adet": 10},
		},
		"fiyatlar": []any{
			map[string]any{"oda_tipi": "standart", "tutar": 85.0, "birim": "oda_gecelik"},
		},
		"release": map[string]any{"gun": 21.0},
		"stop_sale": []any{
			map[string]any{"baslangic": "2026-07-10", "bitis": "2026-07-18"},
		},
		"cikarim_meta": map[string]any{
			"donem":             map[string]any{"guven": 0.9},
			"oda_kontenjanlari": map[string]any{"guven": 0.9},
			"fiyatlar":          map[string]any{"guven": 0.9},
			"release":           map[string]any{"guven": 0.9},
			"stop_sale":         map[string]any{"guven": 0.9},
		},
	}
}

func hasCode(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func TestRuleDateConflict(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		data := validContract()
		data["donem"].(map[string]any)["baslangic"] = "2026-05-01"
		data["donem"].(map[string]any)["bitis"] = "2026-04-20"
		got := ruleDateConflict(data)
		if len(got) != 1 || got[0].Code != CodeDateConflict {
			t.Fatalf("beklenen R1, geldi %#v", got)
		}
		if got[0].Title != "Çelişkili sözleşme tarihi" {
			t.Fatalf("başlık = %q", got[0].Title)
		}
		if got[0].Severity != SeverityCritical {
			t.Fatalf("önem = %s", got[0].Severity)
		}
	})
	t.Run("negative", func(t *testing.T) {
		if got := ruleDateConflict(validContract()); len(got) != 0 {
			t.Fatalf("geçerli dönemde R1 olmamalı: %#v", got)
		}
	})
}

func TestRuleSubperiodOverflow(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		data := validContract()
		data["donem"].(map[string]any)["alt_donemler"] = []any{
			map[string]any{"ad": "Erken", "baslangic": "2026-03-01", "bitis": "2026-05-01"},
		}
		got := ruleSubperiodOverflow(data)
		if len(got) != 1 || got[0].Code != CodeSubperiodOverflow {
			t.Fatalf("beklenen R2, geldi %#v", got)
		}
		if got[0].Severity != SeverityWarning {
			t.Fatalf("önem = %s", got[0].Severity)
		}
	})
	t.Run("negative", func(t *testing.T) {
		data := validContract()
		data["donem"].(map[string]any)["alt_donemler"] = []any{
			map[string]any{"ad": "Yüksek", "baslangic": "2026-06-01", "bitis": "2026-08-31"},
		}
		if got := ruleSubperiodOverflow(data); len(got) != 0 {
			t.Fatalf("içerideki alt dönemde R2 olmamalı: %#v", got)
		}
	})
}

func TestRulePriceAllotmentMismatch(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		data := validContract()
		data["fiyatlar"] = []any{
			map[string]any{"oda_tipi": "standart", "tutar": 85.0},
			map[string]any{"oda_tipi": "junior suite", "tutar": 999.0},
			map[string]any{"oda_tipi": "penthouse", "tutar": 1500.0},
		}
		got := rulePriceAllotmentMismatch(data)
		if len(got) != 2 {
			t.Fatalf("2 R3 beklenirdi, geldi %#v", got)
		}
		for _, f := range got {
			if f.Code != CodePriceAllotmentMismatch {
				t.Fatalf("kod = %s", f.Code)
			}
			if f.Title != "Fiyat–kontenjan uyuşmazlığı" {
				t.Fatalf("başlık = %q", f.Title)
			}
			if !strings.Contains(f.Description, "oda_tipi") {
				t.Fatalf("açıklama alan adını içermeli: %q", f.Description)
			}
			if strings.Contains(f.Description, "999") || strings.Contains(f.Description, "1500") {
				t.Fatalf("açıklama fiyat değeri içermemeli: %q", f.Description)
			}
		}
	})
	t.Run("negative", func(t *testing.T) {
		if got := rulePriceAllotmentMismatch(validContract()); len(got) != 0 {
			t.Fatalf("eşleşen oda tipinde R3 olmamalı: %#v", got)
		}
	})
}

func TestRuleMissingStopSale(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		data := validContract()
		data["stop_sale"] = []any{}
		got := ruleMissingStopSale(data)
		if len(got) != 1 || got[0].Code != CodeMissingStopSale {
			t.Fatalf("beklenen R4, geldi %#v", got)
		}
		if got[0].Title != "Stop-sale maddesi yok" {
			t.Fatalf("başlık = %q", got[0].Title)
		}
		if got[0].Severity != SeverityWarning {
			t.Fatalf("önem = %s", got[0].Severity)
		}
	})
	t.Run("negative", func(t *testing.T) {
		if got := ruleMissingStopSale(validContract()); len(got) != 0 {
			t.Fatalf("dolu stop_sale'de R4 olmamalı: %#v", got)
		}
	})
}

func TestRuleReleaseUnreasonable(t *testing.T) {
	t.Run("positive_zero", func(t *testing.T) {
		data := validContract()
		data["release"].(map[string]any)["gun"] = 0
		got := ruleReleaseUnreasonable(data)
		if len(got) != 1 || got[0].Code != CodeReleaseUnreasonable {
			t.Fatalf("gun=0 için R5 beklenirdi: %#v", got)
		}
	})
	t.Run("positive_over", func(t *testing.T) {
		data := validContract()
		data["release"].(map[string]any)["gun"] = 91
		got := ruleReleaseUnreasonable(data)
		if len(got) != 1 {
			t.Fatalf("gun=91 için R5 beklenirdi: %#v", got)
		}
	})
	t.Run("negative", func(t *testing.T) {
		data := validContract()
		data["release"].(map[string]any)["gun"] = 90
		if got := ruleReleaseUnreasonable(data); len(got) != 0 {
			t.Fatalf("gun=90 için R5 olmamalı: %#v", got)
		}
	})
}

func TestRuleMissingRequired(t *testing.T) {
	t.Run("positive_empty", func(t *testing.T) {
		data := validContract()
		data["donem"] = map[string]any{"baslangic": nil, "bitis": nil}
		got := ruleMissingRequired(data)
		if !hasCode(got, CodeMissingRequired) {
			t.Fatalf("boş dönem için R6 beklenirdi: %#v", got)
		}
		found := false
		for _, f := range got {
			if f.FieldPath == "donem" {
				found = true
			}
		}
		if !found {
			t.Fatalf("donem yolu yok: %#v", got)
		}
	})
	t.Run("positive_confidence", func(t *testing.T) {
		data := validContract()
		data["cikarim_meta"].(map[string]any)["fiyatlar"] = map[string]any{"guven": 0.2}
		got := ruleMissingRequired(data)
		found := false
		for _, f := range got {
			if f.Code == CodeMissingRequired && f.FieldPath == "fiyatlar" {
				found = true
			}
		}
		if !found {
			t.Fatalf("güven 0.2 için R6 beklenirdi: %#v", got)
		}
	})
	t.Run("negative", func(t *testing.T) {
		if got := ruleMissingRequired(validContract()); len(got) != 0 {
			t.Fatalf("dolu çekirdek alanlarda R6 olmamalı: %#v", got)
		}
	})
}

func TestRunRules_CoralDump(t *testing.T) {
	raw := `
{"meta": {"otel_adi": "Belek Palace Hotel", "acente_adi": "Belek Turizm", "para_birimi": "EUR", "kur_esasi": "sabit_kur", "yetkili_mahkeme": "Antalya", "sozlesme_tipi": "YETKİLİ_SÖZLEŞME", "sezon": "yaz"}, "donem": {"baslangic": "2026-05-01", "bitis": "2026-04-20", "alt_donemler": []}}, "oda_kontenjanlari": [{"oda_tipi": "standard", "adet": 150}], "fiyatlar": [{"oda_tipi": "standard", "tutar": 65.0, "birim": "oda_gecelik", "pansiyon": "RO", "release": {"gun": 10, "kapsam": "isim_listesi"}}}, {"oda_tipi": "family", "tutar": 98.0, "birim": "oda_gecelik", "pansiyon": "BB", "release": {"gun": 10, "kapsam": "isim_listesi"}}}]}, "release": {"gun": 10, "kapsam": "isim_listesi"}, "stop_sale": []}}`
	data, repairs, errs := decodeAndCheck(raw)
	if len(errs) != 0 {
		t.Fatalf("şema: %v repairs=%v", errs, repairs)
	}
	got := runRules(data)
	if !hasCode(got, CodeDateConflict) {
		t.Fatalf("coral dump R1 yok: %#v donem=%v", got, data["donem"])
	}
	if !hasCode(got, CodePriceAllotmentMismatch) {
		t.Fatalf("coral dump R3 yok: %#v fiyatlar=%v kontenjan=%v", got, data["fiyatlar"], data["oda_kontenjanlari"])
	}
	if !hasCode(got, CodeMissingStopSale) {
		t.Fatalf("coral dump R4 yok: %#v", got)
	}
}

func TestRunRules_CoralShape(t *testing.T) {
	data := map[string]any{
		"donem": map[string]any{"baslangic": "2026-05-01", "bitis": "2026-04-20"},
		"oda_kontenjanlari": []any{
			map[string]any{"oda_tipi": "standart", "adet": 20},
			map[string]any{"oda_tipi": "aile", "adet": 8},
		},
		"fiyatlar": []any{
			map[string]any{"oda_tipi": "standart", "tutar": 90.0},
			map[string]any{"oda_tipi": "junior suite", "tutar": 140.0},
			map[string]any{"oda_tipi": "penthouse", "tutar": 220.0},
		},
		"release":   map[string]any{"gun": 10.0},
		"stop_sale": []any{},
		"cikarim_meta": map[string]any{
			"donem":             map[string]any{"guven": 0.9},
			"oda_kontenjanlari": map[string]any{"guven": 0.9},
			"fiyatlar":          map[string]any{"guven": 0.9},
			"release":           map[string]any{"guven": 0.9},
			"stop_sale":         map[string]any{"guven": 0.9},
		},
	}
	got := runRules(data)
	if !hasCode(got, CodeDateConflict) {
		t.Fatal("coral: R1 beklenirdi")
	}
	if !hasCode(got, CodePriceAllotmentMismatch) {
		t.Fatal("coral: R3 beklenirdi")
	}
	if !hasCode(got, CodeMissingStopSale) {
		t.Fatal("coral: R4 beklenirdi")
	}
	if hasCode(got, CodeReleaseUnreasonable) {
		t.Fatal("coral: release 10 gün R5 tetiklememeli")
	}
}
