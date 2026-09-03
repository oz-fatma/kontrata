package service

import (
	"context"
	"errors"
	"testing"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/agent"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

func TestApplyAuditOutcome_ErrorDoesNotSetHata(t *testing.T) {
	doc := &repository.Contract{Status: string(model.SozlesmeDurumuIsleniyor)}
	extract := &agent.ExtractResult{
		Data: map[string]any{
			"donem": map[string]any{"baslangic": "2026-04-01", "bitis": "2026-10-31"},
		},
	}
	applyExtract(doc, extract)
	applyAuditOutcome(doc, nil, errors.New("denetci bağlantısı koptu"))
	doc.Status = string(model.SozlesmeDurumuIncelenmeyiBekliyor)

	if doc.Status == string(model.SozlesmeDurumuHata) {
		t.Fatal("denetçi hatası sözleşmeyi HATA yapmamalı")
	}
	if doc.Status != string(model.SozlesmeDurumuIncelenmeyiBekliyor) {
		t.Fatalf("durum = %s", doc.Status)
	}
	if len(doc.Findings) != 0 {
		t.Fatalf("hata durumunda bulgu yazılmamalı: %#v", doc.Findings)
	}
}

func TestApplyAuditOutcome_WritesFindings(t *testing.T) {
	doc := &repository.Contract{}
	res := &agent.AuditResult{
		Findings: []agent.Finding{{
			Code:        agent.CodeDateConflict,
			Title:       "Çelişkili sözleşme tarihi",
			Description: "Tarihleri düzeltin.",
			Severity:    agent.SeverityCritical,
			Source:      agent.SourceRule,
			FieldPath:   "donem/bitis",
		}},
	}
	applyAuditOutcome(doc, res, nil)
	if len(doc.Findings) != 1 {
		t.Fatalf("bulgu sayısı = %d", len(doc.Findings))
	}
	if doc.Findings[0].Code != agent.CodeDateConflict {
		t.Fatalf("kod = %s", doc.Findings[0].Code)
	}
	if doc.AuditorSeconds == nil {
		t.Fatal("denetciSuresi yazılmalı")
	}
}

type stubLLM struct {
	out string
	err error
}

func (s *stubLLM) Generate(context.Context, string, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.out == "" {
		return "[]", nil
	}
	return s.out, nil
}

func TestRunAudit_CoralWritesFindings(t *testing.T) {
	s := &ContractService{llm: &stubLLM{out: "[]"}}
	doc := &repository.Contract{}
	data := map[string]any{
		"donem": map[string]any{"baslangic": "2026-05-01", "bitis": "2026-04-20"},
		"oda_kontenjanlari": []any{
			map[string]any{"oda_tipi": "standard", "adet": 150},
		},
		"fiyatlar": []any{
			map[string]any{"oda_tipi": "standard", "tutar": 65.0, "birim": "oda_gecelik"},
			map[string]any{"oda_tipi": "family", "tutar": 98.0, "birim": "oda_gecelik"},
		},
		"release":   map[string]any{"gun": 10.0},
		"stop_sale": []any{},
	}
	s.runAudit(context.Background(), doc, data, []string{"approximately 10 days"})
	codes := map[string]bool{}
	for _, f := range doc.Findings {
		codes[f.Code] = true
	}
	for _, want := range []string{
		agent.CodeDateConflict,
		agent.CodePriceAllotmentMismatch,
		agent.CodeMissingStopSale,
	} {
		if !codes[want] {
			t.Fatalf("eksik kural %s: %#v", want, doc.Findings)
		}
	}
	if doc.AuditorSeconds == nil {
		t.Fatal("denetciSuresi yazılmalı")
	}
	out := toModel(doc)
	if len(out.Bulgular) < 3 {
		t.Fatalf("GraphQL bulgular = %d", len(out.Bulgular))
	}
}
