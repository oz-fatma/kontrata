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

func postResetRequest(t *testing.T, c *client.Client, eposta string) bool {
	t.Helper()
	var out struct{ SifreSifirlamaIste bool }
	c.MustPost(`mutation ($e: String!) { sifreSifirlamaIste(eposta: $e) }`, &out, client.Var("e", eposta))
	return out.SifreSifirlamaIste
}

func postResetPassword(t *testing.T, c *client.Client, token, sifre string) (bool, error) {
	t.Helper()
	var out struct{ SifreSifirla bool }
	err := c.Post(`mutation ($t: String!, $s: String!) {
		sifreSifirla(token: $t, yeniSifre: $s)
	}`, &out, client.Var("t", token), client.Var("s", sifre))
	return out.SifreSifirla, err
}

func TestPasswordResetExpiredTokenRejected(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	postRegister(t, env.c, eposta, testPassword)
	if !postResetRequest(t, env.c, eposta) {
		t.Fatal("sıfırlama isteği false döndü")
	}
	plain := tokenAfterLabel(env.mail.lastBody(), "Sıfırlama kodunuz:")
	if plain == "" {
		t.Fatal("sıfırlama kodu yok")
	}
	doc, err := env.tokens.GetByHash(ctx, auth.HashToken(plain))
	if err != nil {
		t.Fatalf("token okunamadı")
	}
	if doc.Purpose != repository.PurposePasswordReset {
		t.Fatalf("amac = %s", doc.Purpose)
	}
	doc.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	if err := env.tokens.Update(ctx, doc); err != nil {
		t.Fatalf("token güncellenemedi")
	}
	ok, err := postResetPassword(t, env.c, plain, yeniSifre)
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if ok {
		t.Fatal("süresi geçmiş kod kabul edildi")
	}
	user, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if err := auth.VerifyPassword(testPassword, user.PasswordHash); err != nil {
		t.Fatal("eski şifre değişmiş")
	}
}

func TestPasswordResetUsedTokenRejected(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	postRegister(t, env.c, eposta, testPassword)
	if !postResetRequest(t, env.c, eposta) {
		t.Fatal("sıfırlama isteği false döndü")
	}
	plain := tokenAfterLabel(env.mail.lastBody(), "Sıfırlama kodunuz:")
	ok, err := postResetPassword(t, env.c, plain, yeniSifre)
	if err != nil || !ok {
		t.Fatalf("ilk sıfırlama başarısız: ok=%v err=%v", ok, err)
	}
	ok, err = postResetPassword(t, env.c, plain, "baska-oniki-kr")
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if ok {
		t.Fatal("kullanılmış kod ikinci kez kabul edildi")
	}
	user, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if err := auth.VerifyPassword(yeniSifre, user.PasswordHash); err != nil {
		t.Fatal("ilk yeni şifre durmuyor")
	}
}

func TestPasswordResetRequestUnknownEmailSameResponse(t *testing.T) {
	_, env := setupRegister(t)
	kayitli := uniqueEmail()
	postRegister(t, env.c, kayitli, testPassword)
	mails := env.mail.count()
	if !postResetRequest(t, env.c, kayitli) {
		t.Fatal("kayıtlı e-posta false döndü")
	}
	if env.mail.count() != mails+1 {
		t.Fatal("kayıtlı e-postaya ileti gitmedi")
	}

	kayitsiz := uniqueEmail()
	if !postResetRequest(t, env.c, kayitsiz) {
		t.Fatal("kayıtsız e-posta farklı yanıt verdi")
	}
	if env.mail.count() != mails+1 {
		t.Fatal("kayıtsız e-postaya ileti gitti")
	}
}

func TestPasswordResetRejectsWeakPassword(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	postRegister(t, env.c, eposta, testPassword)
	if !postResetRequest(t, env.c, eposta) {
		t.Fatal("sıfırlama isteği false döndü")
	}
	plain := tokenAfterLabel(env.mail.lastBody(), "Sıfırlama kodunuz:")
	ok, err := postResetPassword(t, env.c, plain, "kisa")
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
	if stored.Used {
		t.Fatal("zayıf şifre kodu tüketti")
	}
	user, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if err := auth.VerifyPassword(testPassword, user.PasswordHash); err != nil {
		t.Fatal("zayıf deneme şifreyi değiştirdi")
	}
}
