package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/oz-fatma/kontrata/backend/internal/llm"
)

func TestAuditLLM_ValidJSON(t *testing.T) {
	stub := &stubLLM{responses: []string{`[{"baslik":"Belirsiz ifade","aciklama":"approximately 10 days kesin bir süre değil.","onem":"UYARI","alan":"release"}]`}}
	a := &Auditor{LLM: stub}
	res, err := a.Audit(context.Background(), validContract(), []string{"Release approximately 10 days prior."})
	if err != nil {
		t.Fatalf("hata beklenmez: %v", err)
	}
	var found Finding
	ok := false
	for _, f := range res.Findings {
		if f.Source == SourceModel {
			found = f
			ok = true
		}
	}
	if !ok {
		t.Fatalf("model bulgusu yok: %#v", res.Findings)
	}
	if found.Title != "Belirsiz ifade" {
		t.Fatalf("başlık = %q", found.Title)
	}
	if found.Severity != SeverityWarning {
		t.Fatalf("önem = %s", found.Severity)
	}
	if found.FieldPath != "release" {
		t.Fatalf("alan = %q", found.FieldPath)
	}
	if stub.calls != 1 {
		t.Fatalf("çağrı = %d", stub.calls)
	}
	if !strings.Contains(stub.users[0], "approximately") {
		t.Fatal("kullanıcı mesajı sözleşme metnini içermeli")
	}
	if strings.Contains(stub.users[0], "cikarim_meta") {
		t.Fatal("LLM'e cikarim_meta gönderilmemeli")
	}
}

func TestAuditLLM_GarbageNoError(t *testing.T) {
	data := validContract()
	data["donem"].(map[string]any)["baslangic"] = "2026-05-01"
	data["donem"].(map[string]any)["bitis"] = "2026-04-20"
	stub := &stubLLM{responses: []string{"bu json değil, düz metin"}}
	a := &Auditor{LLM: stub}
	res, err := a.Audit(context.Background(), data, []string{"metin"})
	if err != nil {
		t.Fatalf("bozuk çıktıda hata dönülmemeli: %v", err)
	}
	if !hasCode(res.Findings, CodeDateConflict) {
		t.Fatalf("kural bulgusu korunmalı: %#v", res.Findings)
	}
	for _, f := range res.Findings {
		if f.Source == SourceModel {
			t.Fatalf("bozuk çıktıda model bulgusu olmamalı: %#v", f)
		}
	}
}

func TestAuditLLM_EmptyArray(t *testing.T) {
	stub := &stubLLM{responses: []string{"[]"}}
	a := &Auditor{LLM: stub}
	res, err := a.Audit(context.Background(), validContract(), nil)
	if err != nil {
		t.Fatalf("hata beklenmez: %v", err)
	}
	for _, f := range res.Findings {
		if f.Source == SourceModel {
			t.Fatalf("boş dizide model bulgusu olmamalı: %#v", res.Findings)
		}
	}
}

func TestParseLLMFindings_MarkdownFence(t *testing.T) {
	raw := "```json\n[{\"baslik\":\"Belirsiz ifade\",\"aciklama\":\"approximately 10 days.\",\"onem\":\"UYARI\",\"alan\":\"release\"}]\n```"
	got := parseLLMFindings(raw)
	if len(got) != 1 {
		t.Fatalf("çit içinden 1 bulgu: %#v", got)
	}
	if got[0].Source != SourceModel || got[0].Title != "Belirsiz ifade" {
		t.Fatalf("ayrıştırma: %#v", got[0])
	}
}

func TestAudit_LLMErrorReturnsRules(t *testing.T) {
	data := validContract()
	data["stop_sale"] = []any{}
	stub := &stubLLM{err: llm.ErrUnavailable}
	a := &Auditor{LLM: stub}
	res, err := a.Audit(context.Background(), data, []string{"sayfa"})
	if err != nil {
		t.Fatalf("LLM hatası Audit hatasına dönüşmemeli: %v", err)
	}
	if !hasCode(res.Findings, CodeMissingStopSale) {
		t.Fatalf("kural bulgusu beklenirdi: %#v", res.Findings)
	}
	for _, f := range res.Findings {
		if f.Source == SourceModel {
			t.Fatalf("LLM hatasında model bulgusu olmamalı")
		}
	}
}

func TestAudit_SortsBySeverity(t *testing.T) {
	data := validContract()
	data["donem"].(map[string]any)["baslangic"] = "2026-05-01"
	data["donem"].(map[string]any)["bitis"] = "2026-04-20"
	data["stop_sale"] = []any{}
	stub := &stubLLM{responses: []string{`[{"baslik":"Bilgi notu","aciklama":"Sektör dışı bir ifade var.","onem":"BILGI","alan":""}]`}}
	a := &Auditor{LLM: stub}
	res, err := a.Audit(context.Background(), data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) < 3 {
		t.Fatalf("en az 3 bulgu: %#v", res.Findings)
	}
	if res.Findings[0].Severity != SeverityCritical {
		t.Fatalf("ilk önem = %s", res.Findings[0].Severity)
	}
	last := res.Findings[len(res.Findings)-1]
	if last.Severity != SeverityInfo {
		t.Fatalf("son önem = %s", last.Severity)
	}
}
