package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	appmongo "github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

type silentMailer struct{}

func (silentMailer) Gonder(_, _, _ string) error { return nil }

func TestHesapSilmeHataGeriAlinir(t *testing.T) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		t.Skip("MONGO_URI yok")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := appmongo.Connect(ctx, uri)
	if err != nil {
		t.Fatalf("mongo bağlanamadı")
	}
	t.Cleanup(func() {
		_ = db.Disconnect(context.Background())
	})
	if !db.ReplicaSet(ctx) {
		t.Skip("hesap silme atomik işlem için replica set gerekli")
	}

	users := repository.NewKullaniciRepository(db)
	tokens := repository.NewDogrulamaTokenRepository(db)
	mfa := repository.NewMFAKoduRepository(db)
	sessions := repository.NewOturumRepository(db)
	devices := repository.NewCihazRepository(db)
	soz := repository.NewSozlesmeRepository(db)
	audit := repository.NewDenetimRepository(db)
	for _, ensure := range []func(context.Context) error{
		users.EnsureIndexes, tokens.EnsureIndexes, mfa.EnsureIndexes,
		sessions.EnsureIndexes, devices.EnsureIndexes, soz.EnsureIndexes, audit.EnsureIndexes,
	} {
		if err := ensure(ctx); err != nil {
			t.Fatalf("indeks: %v", err)
		}
	}
	signer, err := auth.NewJWT([]byte("test-jwt-secret-at-least-32-bytes!!"))
	if err != nil {
		t.Fatal(err)
	}
	svc := NewAuthService(users, tokens, mfa, sessions, devices, soz, audit, silentMailer{}, auth.Params{Time: 1, Memory: 8 * 1024, Threads: 1, KeyLen: 32, SaltLen: 16}, signer, db)

	hash, err := auth.HashPassword("oniki-karakter", svc.params)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	user := repository.Kullanici{
		Eposta:           "silme-" + bson.NewObjectID().Hex() + "@ornek.test",
		SifreHash:        hash,
		EpostaDogrulandi: true,
		Durum:            repository.DurumAktif,
		OlusturmaTarihi:  now,
		GuncellemeTarihi: now,
	}
	if err := users.Create(ctx, &user); err != nil {
		t.Fatal(err)
	}
	if err := soz.Create(ctx, &repository.Sozlesme{
		KullaniciID:      user.ID,
		OlusturmaTarihi:  now,
		GuncellemeTarihi: now,
		Durum:            "YUKLENDI",
	}); err != nil {
		t.Fatal(err)
	}
	if err := devices.Create(ctx, &repository.Cihaz{
		KullaniciID:    user.ID,
		CihazParmakIzi: "abc",
		Ad:             "test",
		IlkGorulme:     now,
		SonGorulme:     now,
	}); err != nil {
		t.Fatal(err)
	}
	plain, tokHash, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := tokens.Create(ctx, &repository.DogrulamaTokeni{
		KullaniciID: user.ID,
		Token:       tokHash,
		Amac:        repository.AmacHesapSilme,
		SonKullanma: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	svc.deleteFailAt = deleteStepUser
	ok, err := svc.HesapSil(ctx, plain)
	if ok || err == nil {
		t.Fatal("hata bekleniyordu")
	}
	if !errors.Is(err, errDeleteProbe) {
		t.Fatalf("err = %v", err)
	}
	if _, err := users.GetByID(ctx, user.ID); err != nil {
		t.Fatal("kullanıcı silinmiş")
	}
	if n, _ := soz.CountByUser(ctx, user.ID); n != 1 {
		t.Fatalf("sözleşme silinmiş n=%d", n)
	}
	if n, _ := devices.CountByUser(ctx, user.ID); n != 1 {
		t.Fatalf("cihaz silinmiş n=%d", n)
	}
	if n, _ := tokens.CountByUser(ctx, user.ID); n != 1 {
		t.Fatalf("token silinmiş n=%d", n)
	}
}
