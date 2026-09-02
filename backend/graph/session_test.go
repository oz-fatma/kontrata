package graph

import (
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
)

func TestLoginAndMFAOpensSession(t *testing.T) {
	_, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	access, refresh := loginSession(t, env, eposta, testPassword)
	if access == "" || refresh == "" {
		t.Fatal("jetonlar boş")
	}
	c := env.withToken(access)
	var out struct {
		Oturumlarim []struct {
			ID       string
			MevcutMu bool
		}
	}
	c.MustPost(`query { oturumlarim { id mevcutMu } }`, &out)
	if len(out.Oturumlarim) != 1 || !out.Oturumlarim[0].MevcutMu {
		t.Fatal("mevcut oturum listelenmedi")
	}
}

func TestMFAFiveWrongCodesInvalidate(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	var giris struct {
		GirisYap struct{ GeciciToken string }
	}
	env.c.MustPost(`mutation ($e: String!, $s: String!) {
		girisYap(eposta: $e, sifre: $s) { geciciToken }
	}`, &giris, client.Var("e", eposta), client.Var("s", testPassword))
	kod := tokenAfterLabel(env.mail.lastBody(), "Giriş kodunuz:")
	gecici := giris.GirisYap.GeciciToken
	wrong := "000000"
	if kod == wrong {
		wrong = "111111"
	}
	for i := 0; i < auth.MFAMaxAttempts; i++ {
		var out struct{ MfaDogrula struct{ ErisimJetonu string } }
		err := env.c.Post(`mutation ($g: String!, $k: String!) {
			mfaDogrula(geciciToken: $g, kod: $k) { erisimJetonu }
		}`, &out, client.Var("g", gecici), client.Var("k", wrong))
		if err == nil {
			t.Fatal("yanlış kod kabul edildi")
		}
	}
	user, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	active, err := env.mfa.GetActive(ctx, user.ID, time.Now().UTC())
	if err == nil && active != nil {
		t.Fatal("5 denemeden sonra kod hâlâ aktif")
	}
	var out struct{ MfaDogrula struct{ ErisimJetonu string } }
	err = env.c.Post(`mutation ($g: String!, $k: String!) {
		mfaDogrula(geciciToken: $g, kod: $k) { erisimJetonu }
	}`, &out, client.Var("g", gecici), client.Var("k", kod))
	if err == nil {
		t.Fatal("geçersizleşen kod doğru değerle kabul edildi")
	}
}

func TestMFAExpiredCodeRejected(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	var giris struct {
		GirisYap struct{ GeciciToken string }
	}
	env.c.MustPost(`mutation ($e: String!, $s: String!) {
		girisYap(eposta: $e, sifre: $s) { geciciToken }
	}`, &giris, client.Var("e", eposta), client.Var("s", testPassword))
	kod := tokenAfterLabel(env.mail.lastBody(), "Giriş kodunuz:")
	user, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	doc, err := env.mfa.GetActive(ctx, user.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("MFA okunamadı")
	}
	doc.ExpiresAt = time.Now().UTC().Add(-time.Second)
	if err := env.mfa.Update(ctx, doc); err != nil {
		t.Fatalf("MFA güncellenemedi")
	}
	var out struct{ MfaDogrula struct{ ErisimJetonu string } }
	err = env.c.Post(`mutation ($g: String!, $k: String!) {
		mfaDogrula(geciciToken: $g, kod: $k) { erisimJetonu }
	}`, &out, client.Var("g", giris.GirisYap.GeciciToken), client.Var("k", kod))
	if err == nil {
		t.Fatal("süresi geçmiş kod kabul edildi")
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	_, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	_, refresh := loginSession(t, env, eposta, testPassword)
	var yenile struct {
		JetonYenile struct {
			ErisimJetonu   string
			YenilemeJetonu string
		}
	}
	env.c.MustPost(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu yenilemeJetonu }
	}`, &yenile, client.Var("r", refresh))
	if yenile.JetonYenile.YenilemeJetonu == "" || yenile.JetonYenile.YenilemeJetonu == refresh {
		t.Fatal("yeni yenileme jetonu yok")
	}
	var tekrar struct {
		JetonYenile struct{ ErisimJetonu string }
	}
	err := env.c.Post(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
	}`, &tekrar, client.Var("r", refresh))
	assertYenilemeReddi(t, err)
	env.c.MustPost(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu yenilemeJetonu }
	}`, &yenile, client.Var("r", yenile.JetonYenile.YenilemeJetonu))
}

func postRefreshToken(t *testing.T, c *client.Client, refresh string) (access, yeniRefresh string) {
	t.Helper()
	var out struct {
		JetonYenile struct {
			ErisimJetonu   string
			YenilemeJetonu string
		}
	}
	if err := c.Post(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu yenilemeJetonu }
	}`, &out, client.Var("r", refresh)); err != nil {
		t.Fatalf("jetonYenile: %v", err)
	}
	if out.JetonYenile.ErisimJetonu == "" || out.JetonYenile.YenilemeJetonu == "" || out.JetonYenile.YenilemeJetonu == refresh {
		t.Fatal("yenileme başarısız")
	}
	return out.JetonYenile.ErisimJetonu, out.JetonYenile.YenilemeJetonu
}

func assertYenilemeReddi(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("geçersiz yenileme jetonu kabul edildi")
	}
	msg := err.Error()
	if !strings.Contains(msg, "oturum sonlandı, tekrar giriş yapın") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(msg, "kimlik doğrulaması gerekli") {
		t.Fatalf("yenileme @auth mesajı döndü: %v", err)
	}
}

func expiredAccess(t *testing.T, access string) string {
	t.Helper()
	j, err := auth.NewJWT([]byte(testJWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	uid, sid, err := j.ParseAccess(access)
	if err != nil {
		t.Fatal(err)
	}
	expired, err := j.SignAccess(uid, sid, time.Now().UTC().Add(-auth.AccessTTL-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := j.ParseAccess(expired); err == nil {
		t.Fatal("süresi dolmuş jeton hâlâ geçerli")
	}
	return expired
}

func TestRefreshDoesNotNeedAccessToken(t *testing.T) {
	_, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	access, refresh := loginSession(t, env, eposta, testPassword)

	_, refresh = postRefreshToken(t, env.c, refresh)

	access, _ = postRefreshToken(t, env.withToken(expiredAccess(t, access)), refresh)

	c := env.withToken(access)
	var out struct {
		Oturumlarim []struct {
			ID       string
			MevcutMu bool
		}
	}
	c.MustPost(`query { oturumlarim { id mevcutMu } }`, &out)
	if len(out.Oturumlarim) != 1 || !out.Oturumlarim[0].MevcutMu {
		t.Fatal("yenilenen oturum kullanılamadı")
	}
}

func TestRefreshInvalidSameMessage(t *testing.T) {
	_, env := setupRegister(t)
	var out struct {
		JetonYenile struct{ ErisimJetonu string }
	}
	errBos := env.c.Post(`mutation { jetonYenile(yenilemeJetonu: "") { erisimJetonu } }`, &out)
	errYok := env.c.Post(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
	}`, &out, client.Var("r", "yok-boyle-bir-jeton"))
	assertYenilemeReddi(t, errBos)
	assertYenilemeReddi(t, errYok)
}

func TestPasswordResetDropsOldSessions(t *testing.T) {
	_, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	access, refresh := loginSession(t, env, eposta, testPassword)
	if !postResetRequest(t, env.c, eposta) {
		t.Fatal("sıfırlama isteği false")
	}
	plain := tokenAfterLabel(env.mail.lastBody(), "Sıfırlama kodunuz:")
	ok, err := postResetPassword(t, env.c, plain, "yeni-oniki-kr")
	if err != nil || !ok {
		t.Fatalf("sıfırlama: ok=%v err=%v", ok, err)
	}
	var tekrar struct {
		JetonYenile struct{ ErisimJetonu string }
	}
	err = env.c.Post(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
	}`, &tekrar, client.Var("r", refresh))
	assertYenilemeReddi(t, err)
	c := env.withToken(access)
	var q struct{ Sozlesmeler []any }
	err = c.Post(`query { sozlesmeler { id } }`, &q)
	if err == nil {
		t.Fatal("iptal oturumla @auth kabul edildi")
	}
	if !strings.Contains(err.Error(), "kimlik doğrulaması gerekli") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoginWrongAccountSameResponse(t *testing.T) {
	_, env := setupRegister(t)
	eposta := uniqueEmail()
	registerVerified(t, env, eposta, testPassword)
	mails := env.mail.count()
	post := func(e, s string) (mfa bool, token string) {
		t.Helper()
		var out struct {
			GirisYap struct {
				MfaGerekli  bool
				GeciciToken string
			}
		}
		env.c.MustPost(`mutation ($e: String!, $s: String!) {
			girisYap(eposta: $e, sifre: $s) { mfaGerekli geciciToken }
		}`, &out, client.Var("e", e), client.Var("s", s))
		return out.GirisYap.MfaGerekli, out.GirisYap.GeciciToken
	}
	mfaYanlis, tokYanlis := post(eposta, "yanlis-sifre-12")
	mfaYok, tokYok := post(uniqueEmail(), testPassword)
	if !mfaYanlis || tokYanlis == "" || !mfaYok || tokYok == "" {
		t.Fatal("yanlış şifre veya olmayan hesap farklı yanıt verdi")
	}
	if env.mail.count() != mails {
		t.Fatal("başarısız girişte MFA kodu gönderildi")
	}
}
