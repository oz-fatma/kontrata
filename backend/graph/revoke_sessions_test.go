package graph

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

func TestSessionWithoutDeviceRevokedOnMigration(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	user, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	access, refresh := loginSession(t, env, eposta, testPassword)

	plain, hash, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	res, err := env.db.Collection(mongo.DatabaseName(), "oturumlar").InsertOne(ctx, bson.M{
		"kullaniciId":       user.ID,
		"yenilemeTokenHash": hash,
		"olusturmaTarihi":   now,
		"sonKullanma":       now.Add(auth.RefreshTTL),
		"iptalEdildi":       false,
		"ipAdresi":          "203.0.113.9",
		"kullaniciAjani":    "eski-istemci",
	})
	if err != nil {
		t.Fatalf("eski oturum yazılamadı: %v", err)
	}
	oid, ok := res.InsertedID.(bson.ObjectID)
	if !ok {
		t.Fatal("eski oturum kimliği yok")
	}
	if err := env.sessions.Create(ctx, &repository.Session{
		UserID:           user.ID,
		RefreshTokenHash: "hash-olmadan-cihaz",
		CreatedAt:        now,
		ExpiresAt:        now.Add(auth.RefreshTTL),
	}); !errors.Is(err, repository.ErrInvalidID) {
		t.Fatalf("cihazId olmadan Create: %v", err)
	}
	if err := env.sessions.RevokeMissingDevice(ctx); err != nil {
		t.Fatalf("geçiş: %v", err)
	}
	eski, err := env.sessions.GetByID(ctx, oid)
	if err != nil {
		t.Fatal("eski oturum okunamadı")
	}
	if !eski.Revoked || eski.RevokeReason != repository.RevokePreDeviceRecord {
		t.Fatalf("iptal=%v neden=%q", eski.Revoked, eski.RevokeReason)
	}

	c := env.withToken(access)
	var q struct {
		Oturumlarim []struct{ ID string }
	}
	c.MustPost(`query { oturumlarim { id } }`, &q)
	if len(q.Oturumlarim) < 1 {
		t.Fatal("cihazlı oturum da iptal edildi")
	}

	var yenile struct {
		JetonYenile struct{ ErisimJetonu string }
	}
	err = env.c.Post(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
	}`, &yenile, client.Var("r", plain))
	assertYenilemeReddi(t, err)
	env.c.MustPost(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
	}`, &yenile, client.Var("r", refresh))
}

func TestRevokeOtherSessions(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	_, refresh1 := loginSession(t, env, eposta, testPassword)
	_, refresh2 := loginSession(t, env, eposta, testPassword)
	access3, refresh3 := loginSession(t, env, eposta, testPassword)

	err := env.c.Post(`mutation { tumOturumlariKapat }`, new(struct{ TumOturumlariKapat int32 }))
	if err == nil || !strings.Contains(err.Error(), "kimlik doğrulaması gerekli") {
		t.Fatalf("@auth: %v", err)
	}

	c := env.withToken(access3)
	var out struct{ TumOturumlariKapat int32 }
	c.MustPost(`mutation { tumOturumlariKapat }`, &out)
	if out.TumOturumlariKapat != 2 {
		t.Fatalf("iptal = %d", out.TumOturumlariKapat)
	}
	kayit, err := env.audit.Latest(ctx, repository.EventAllSessionsRevoked)
	if err != nil {
		t.Fatal("TUM_OTURUMLAR_KAPATILDI yok")
	}
	if kayit.Detail != "2" {
		t.Fatalf("detay = %q", kayit.Detail)
	}

	var list struct {
		Oturumlarim []struct {
			ID       string
			MevcutMu bool
		}
	}
	c.MustPost(`query { oturumlarim { id mevcutMu } }`, &list)
	if len(list.Oturumlarim) != 1 || !list.Oturumlarim[0].MevcutMu {
		t.Fatal("mevcut oturum kapanmış")
	}

	var yenile struct {
		JetonYenile struct{ ErisimJetonu string }
	}
	err = env.c.Post(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
	}`, &yenile, client.Var("r", refresh1))
	assertYenilemeReddi(t, err)
	err = env.c.Post(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
	}`, &yenile, client.Var("r", refresh2))
	assertYenilemeReddi(t, err)
	env.c.MustPost(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
	}`, &yenile, client.Var("r", refresh3))
}
