package graph

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

func TestApproveContract(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	access, _ := loginSession(t, env, eposta, testPassword)
	c := env.withToken(access)

	girdi := sozlesmeGirdi()
	girdi["durum"] = "INCELENMEYI_BEKLIYOR"
	var created struct {
		SozlesmeOlustur struct{ ID, Durum string }
	}
	c.MustPost(`mutation ($g: SozlesmeGirdi!) { sozlesmeOlustur(girdi: $g) { id durum } }`,
		&created, client.Var("g", girdi))
	id := created.SozlesmeOlustur.ID
	if id == "" {
		t.Fatal("id boş")
	}
	t.Cleanup(func() {
		_ = env.soz.Delete(context.Background(), id)
	})
	if created.SozlesmeOlustur.Durum != "INCELENMEYI_BEKLIYOR" {
		t.Fatalf("oluşturma durum = %s", created.SozlesmeOlustur.Durum)
	}

	var onay struct {
		SozlesmeOnayla struct{ ID, Durum string }
	}
	c.MustPost(`mutation ($id: ID!) { sozlesmeOnayla(id: $id) { id durum } }`,
		&onay, client.Var("id", id))
	if onay.SozlesmeOnayla.Durum != "ONAYLANDI" {
		t.Fatalf("durum = %s", onay.SozlesmeOnayla.Durum)
	}
	rec, err := env.audit.Latest(ctx, repository.EventContractApproved)
	if err != nil {
		t.Fatalf("denetim yok: %v", err)
	}
	if rec.Event != repository.EventContractApproved {
		t.Fatalf("olay = %s", rec.Event)
	}

	again := c.Post(`mutation ($id: ID!) { sozlesmeOnayla(id: $id) { id durum } }`,
		new(struct{ SozlesmeOnayla struct{ ID string } }), client.Var("id", id))
	if again == nil || !strings.Contains(again.Error(), "yalnızca incelenmeyi bekleyen sözleşme onaylanır") {
		t.Fatalf("ikinci onay: %v", again)
	}

	editErr := c.Post(`mutation ($id: ID!, $a: String!, $d: JSON!) {
		sozlesmeAlanGuncelle(id: $id, alanYolu: $a, deger: $d) { id }
	}`, new(struct{ SozlesmeAlanGuncelle struct{ ID string } }),
		client.Var("id", id), client.Var("a", "meta.otelAdi"), client.Var("d", "Yeni"))
	if editErr == nil || !strings.Contains(editErr.Error(), "onaylı sözleşme düzenlenemez") {
		t.Fatalf("onaylı düzenleme: %v", editErr)
	}
}

func TestUpdateFieldSetsConfidence(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	access, _ := loginSession(t, env, eposta, testPassword)
	c := env.withToken(access)

	girdi := sozlesmeGirdi()
	girdi["durum"] = "INCELENMEYI_BEKLIYOR"
	girdi["meta"] = map[string]any{"otelAdi": "Eski Otel"}
	girdi["cikarimMeta"] = []map[string]any{
		{"alanYolu": "meta.otelAdi", "guven": 0.4},
	}
	var created struct {
		SozlesmeOlustur struct{ ID string }
	}
	c.MustPost(`mutation ($g: SozlesmeGirdi!) { sozlesmeOlustur(girdi: $g) { id } }`,
		&created, client.Var("g", girdi))
	id := created.SozlesmeOlustur.ID
	t.Cleanup(func() {
		_ = env.soz.Delete(context.Background(), id)
	})

	var upd struct {
		SozlesmeAlanGuncelle struct {
			Meta        *struct{ OtelAdi *string }
			CikarimMeta []struct {
				AlanYolu       string
				Guven          *float64
				ElleDuzeltildi *bool
			}
		}
	}
	c.MustPost(`mutation ($id: ID!, $a: String!, $d: JSON!) {
		sozlesmeAlanGuncelle(id: $id, alanYolu: $a, deger: $d) {
			meta { otelAdi }
			cikarimMeta { alanYolu guven elleDuzeltildi }
		}
	}`, &upd, client.Var("id", id), client.Var("a", "meta.otelAdi"), client.Var("d", "Yeni Otel"))
	if upd.SozlesmeAlanGuncelle.Meta == nil || upd.SozlesmeAlanGuncelle.Meta.OtelAdi == nil ||
		*upd.SozlesmeAlanGuncelle.Meta.OtelAdi != "Yeni Otel" {
		t.Fatalf("otel adı = %+v", upd.SozlesmeAlanGuncelle.Meta)
	}
	found := false
	for _, m := range upd.SozlesmeAlanGuncelle.CikarimMeta {
		if m.AlanYolu != "meta.otelAdi" {
			continue
		}
		found = true
		if m.Guven == nil || *m.Guven != 1 {
			t.Fatalf("güven = %v", m.Guven)
		}
		if m.ElleDuzeltildi == nil || !*m.ElleDuzeltildi {
			t.Fatal("elle düzeltildi işareti yok")
		}
	}
	if !found {
		t.Fatal("cikarimMeta satırı yok")
	}
	rec, err := env.audit.Latest(ctx, repository.EventContractFieldUpdated)
	if err != nil {
		t.Fatalf("denetim yok: %v", err)
	}
	if rec.Detail != "meta.otelAdi" {
		t.Fatalf("detay = %q (değer yazılmamalı)", rec.Detail)
	}
	if strings.Contains(rec.Detail, "Yeni Otel") {
		t.Fatal("denetimde alan değeri var")
	}
}

func TestServeFile_OtherOrganization404(t *testing.T) {
	ctx, env := setupRegister(t)
	epostaA := uniqueEmail()
	epostaB := uniqueEmail()
	postRegisterCorporate(t, env.c, epostaA, testPassword, "Akdeniz Otelcilik")
	if !postVerify(t, env.c, tokenFromBody(env.mail.lastBody())) {
		t.Fatal("A doğrulanamadı")
	}
	postRegisterCorporate(t, env.c, epostaB, testPassword, "Ege Turizm")
	if !postVerify(t, env.c, tokenFromBody(env.mail.lastBody())) {
		t.Fatal("B doğrulanamadı")
	}

	accessA, _ := loginSession(t, env, epostaA, testPassword)
	accessB, _ := loginSession(t, env, epostaB, testPassword)
	userA, err := env.users.GetByEmail(ctx, epostaA)
	if err != nil {
		t.Fatalf("kullanıcı A: %v", err)
	}
	stored, err := env.files.Save(bytes.NewReader([]byte("%PDF-1.4 dummy")))
	if err != nil {
		t.Fatalf("pdf: %v", err)
	}
	name := "ornek.pdf"
	now := time.Now().UTC()
	doc := repository.Contract{
		UserID:         userA.ID,
		OrganizationID: userA.OrganizationID,
		CreatedAt:      now,
		UpdatedAt:      now,
		Status:         "YUKLENDI",
		FileName:       &name,
		StoredFileID:   stored,
		RoomAllotments: []repository.RoomAllotment{},
		Prices:         []repository.Price{},
		StopSale:       []repository.StopSaleRange{},
	}
	if err := env.soz.Create(ctx, &doc); err != nil {
		t.Fatalf("kayıt: %v", err)
	}
	t.Cleanup(func() {
		_ = env.soz.Delete(context.Background(), doc.ID.Hex())
	})
	id := doc.ID.Hex()
	if id == "" || id == bson.NilObjectID.Hex() {
		t.Fatal("sözleşme id boş")
	}

	reqB, err := http.NewRequest(http.MethodGet, "/dosya/"+id, nil)
	if err != nil {
		t.Fatal(err)
	}
	reqB.Header.Set("Authorization", "Bearer "+accessB)
	rrB := httptest.NewRecorder()
	env.h.ServeHTTP(rrB, reqB)
	if rrB.Code != http.StatusNotFound {
		t.Fatalf("yabancı org durum = %d gövde=%s", rrB.Code, rrB.Body.String())
	}

	reqA, err := http.NewRequest(http.MethodGet, "/dosya/"+id, nil)
	if err != nil {
		t.Fatal(err)
	}
	reqA.Header.Set("Authorization", "Bearer "+accessA)
	rrA := httptest.NewRecorder()
	env.h.ServeHTTP(rrA, reqA)
	if rrA.Code != http.StatusOK {
		t.Fatalf("sahip durum = %d gövde=%s", rrA.Code, rrA.Body.String())
	}
	if !strings.Contains(rrA.Header().Get("Content-Type"), "pdf") {
		t.Fatalf("content-type = %s", rrA.Header().Get("Content-Type"))
	}
}

func TestApproveRequiresPendingReview(t *testing.T) {
	_, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	access, _ := loginSession(t, env, eposta, testPassword)
	c := env.withToken(access)
	var created struct {
		SozlesmeOlustur struct{ ID string }
	}
	c.MustPost(`mutation ($g: SozlesmeGirdi!) { sozlesmeOlustur(girdi: $g) { id } }`,
		&created, client.Var("g", sozlesmeGirdi()))
	t.Cleanup(func() {
		_ = env.soz.Delete(context.Background(), created.SozlesmeOlustur.ID)
	})
	err := c.Post(`mutation ($id: ID!) { sozlesmeOnayla(id: $id) { id } }`,
		new(struct{ SozlesmeOnayla struct{ ID string } }), client.Var("id", created.SozlesmeOlustur.ID))
	if err == nil || !strings.Contains(err.Error(), "yalnızca incelenmeyi bekleyen sözleşme onaylanır") {
		t.Fatalf("yüklendi onay: %v", err)
	}
}
