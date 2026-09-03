package service

import (
	"testing"

	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

func TestActor_ViewerCannotApprove(t *testing.T) {
	a := actor{user: repository.User{Role: repository.RoleViewer}}
	if a.can(opContractApprove) {
		t.Fatal("görüntüleyici onaylamamalı")
	}
	if a.can(opContractWrite) {
		t.Fatal("görüntüleyici yazmamalı")
	}
	admin := actor{user: repository.User{Role: repository.RoleAdmin}}
	if !admin.can(opContractApprove) {
		t.Fatal("yönetici onaylayabilmeli")
	}
	owner := actor{user: repository.User{Role: repository.RoleOwner}}
	if !owner.can(opContractApprove) {
		t.Fatal("sahip onaylayabilmeli")
	}
}

func TestApplyFieldValue_SetsConfidenceAndFlag(t *testing.T) {
	low := 0.41
	old := "Eski Otel"
	doc := &repository.Contract{
		Meta: &repository.ContractMeta{HotelName: &old},
		ExtractionMeta: []repository.ExtractionMeta{
			{FieldPath: "meta.otelAdi", Confidence: &low},
		},
	}
	path, err := applyFieldValue(doc, "meta.otelAdi", "Yeni Otel")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	markFieldManuallyFixed(doc, path)
	if doc.Meta == nil || doc.Meta.HotelName == nil || *doc.Meta.HotelName != "Yeni Otel" {
		t.Fatalf("otel = %+v", doc.Meta)
	}
	if len(doc.ExtractionMeta) != 1 {
		t.Fatalf("meta uzunluğu = %d", len(doc.ExtractionMeta))
	}
	m := doc.ExtractionMeta[0]
	if m.Confidence == nil || *m.Confidence != 1 {
		t.Fatalf("güven = %v", m.Confidence)
	}
	if !m.ManuallyFixed {
		t.Fatal("elle düzeltildi yok")
	}
}
