package graph

import (
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const yeniSifre = "yeni-oniki-kr"

func postSifirlamaIste(t *testing.T, c *client.Client, eposta string) bool {
	t.Helper()
	var out struct{ SifreSifirlamaIste bool }
	c.MustPost(`mutation ($e: String!) { sifreSifirlamaIste(eposta: $e) }`, &out, client.Var("e", eposta))
	return out.SifreSifirlamaIste
}

func postSifirla(t *testing.T, c *client.Client, token, sifre string) (bool, error) {
	t.Helper()
	var out struct{ SifreSifirla bool }
	err := c.Post(`mutation ($t: String!, $s: String!) {
		sifreSifirla(token: $t, yeniSifre: $s)
	}`, &out, client.Var("t", token), client.Var("s", sifre))
	return out.SifreSifirla, err
}

func TestSifreSifirlamaSuresiGecmisTokenReddedilir(t *testing.T) {
	ctx, env := setupKayit(t)
	eposta := uniqueEposta()
	postKayit(t, env.c, eposta, testSifre)
	if !postSifirlamaIste(t, env.c, eposta) {
		t.Fatal("sıfırlama isteği false döndü")
	}
	plain := tokenAfterLabel(env.mail.lastGovde(), "Sıfırlama kodunuz:")
	if plain == "" {
		t.Fatal("sıfırlama kodu yok")
	}
	doc, err := env.tokens.GetByHash(ctx, auth.HashToken(plain))
	if err != nil {
		t.Fatalf("token okunamadı")
	}
	if doc.Amac != repository.AmacSifreSifirlama {
		t.Fatalf("amac = %s", doc.Amac)
	}
	doc.SonKullanma = time.Now().UTC().Add(-time.Minute)
	if err := env.tokens.Update(ctx, doc); err != nil {
		t.Fatalf("token güncellenemedi")
	}
	ok, err := postSifirla(t, env.c, plain, yeniSifre)
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if ok {
		t.Fatal("süresi geçmiş kod kabul edildi")
	}
	user, err := env.users.GetByEposta(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if err := auth.VerifyPassword(testSifre, user.SifreHash); err != nil {
		t.Fatal("eski şifre değişmiş")
	}
}

func TestSifreSifirlamaKullanilmisTokenReddedilir(t *testing.T) {
	ctx, env := setupKayit(t)
	eposta := uniqueEposta()
	postKayit(t, env.c, eposta, testSifre)
	if !postSifirlamaIste(t, env.c, eposta) {
		t.Fatal("sıfırlama isteği false döndü")
	}
	plain := tokenAfterLabel(env.mail.lastGovde(), "Sıfırlama kodunuz:")
	ok, err := postSifirla(t, env.c, plain, yeniSifre)
	if err != nil || !ok {
		t.Fatalf("ilk sıfırlama başarısız: ok=%v err=%v", ok, err)
	}
	ok, err = postSifirla(t, env.c, plain, "baska-oniki-kr")
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if ok {
		t.Fatal("kullanılmış kod ikinci kez kabul edildi")
	}
	user, err := env.users.GetByEposta(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if err := auth.VerifyPassword(yeniSifre, user.SifreHash); err != nil {
		t.Fatal("ilk yeni şifre durmuyor")
	}
}

func TestSifreSifirlamaIsteKayitsizAyniYanit(t *testing.T) {
	_, env := setupKayit(t)
	kayitli := uniqueEposta()
	postKayit(t, env.c, kayitli, testSifre)
	mails := env.mail.count()
	if !postSifirlamaIste(t, env.c, kayitli) {
		t.Fatal("kayıtlı e-posta false döndü")
	}
	if env.mail.count() != mails+1 {
		t.Fatal("kayıtlı e-postaya ileti gitmedi")
	}

	kayitsiz := uniqueEposta()
	if !postSifirlamaIste(t, env.c, kayitsiz) {
		t.Fatal("kayıtsız e-posta farklı yanıt verdi")
	}
	if env.mail.count() != mails+1 {
		t.Fatal("kayıtsız e-postaya ileti gitti")
	}
}

func TestSifreSifirlaZayifSifreReddedilir(t *testing.T) {
	ctx, env := setupKayit(t)
	eposta := uniqueEposta()
	postKayit(t, env.c, eposta, testSifre)
	if !postSifirlamaIste(t, env.c, eposta) {
		t.Fatal("sıfırlama isteği false döndü")
	}
	plain := tokenAfterLabel(env.mail.lastGovde(), "Sıfırlama kodunuz:")
	ok, err := postSifirla(t, env.c, plain, "kisa")
	if err == nil {
		t.Fatal("zayıf şifre kabul edildi")
	}
	if !strings.Contains(err.Error(), "şifre en az 12") {
		t.Fatalf("err = %v", err)
	}
	if ok {
		t.Fatal("zayıf şifre true döndü")
	}
	stored, err := env.tokens.GetByHash(ctx, auth.HashToken(plain))
	if err != nil {
		t.Fatalf("token okunamadı")
	}
	if stored.Kullanildi {
		t.Fatal("zayıf şifre kodu tüketti")
	}
	user, err := env.users.GetByEposta(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if err := auth.VerifyPassword(testSifre, user.SifreHash); err != nil {
		t.Fatal("zayıf deneme şifreyi değiştirdi")
	}
}
