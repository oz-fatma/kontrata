package mailer

import (
	"strings"
	"testing"
)

func testAppURL(t *testing.T) {
	t.Helper()
	SetAppURL("https://app.ornek.test")
	t.Cleanup(func() { SetAppURL(defaultAppURL) })
}

func assertActionMail(t *testing.T, body, path, codeLabel, token string) {
	t.Helper()
	if !strings.Contains(body, "https://") {
		t.Fatalf("https bağlantısı yok:\n%s", body)
	}
	wantLink := TokenLink(path, token)
	if !strings.Contains(body, wantLink) {
		t.Fatalf("beklenen bağlantı yok %q:\n%s", wantLink, body)
	}
	if !strings.Contains(body, "Aşağıdaki bağlantıya tıklayarak") {
		t.Fatal("bağlantı vurgusu yok")
	}
	if !strings.Contains(body, codeLabel) {
		t.Fatalf("kod etiketi yok %q", codeLabel)
	}
	if !strings.Contains(body, token) {
		t.Fatal("yedek kod metinde yok")
	}
}

func TestVerificationBody(t *testing.T) {
	testAppURL(t)
	const token = "abc-verify-token"
	body := VerificationBody(token)
	assertActionMail(t, body, "/dogrula", "Doğrulama kodunuz:", token)
}

func TestPasswordResetBody(t *testing.T) {
	testAppURL(t)
	const token = "abc-reset-token"
	body := PasswordResetBody(token)
	assertActionMail(t, body, "/sifre-sifirla", "Sıfırlama kodunuz:", token)
}

func TestAccountDeleteBody(t *testing.T) {
	testAppURL(t)
	const token = "abc-delete-token"
	body := AccountDeleteBody(token)
	assertActionMail(t, body, "/ayarlar/", "Hesap silme onay kodunuz:", token)
}

func TestInviteBody(t *testing.T) {
	testAppURL(t)
	const token = "abc-invite-token"
	body := InviteBody(token)
	assertActionMail(t, body, "/kayit", "Davet kodunuz:", token)
}

func TestTokenLink_DefaultAppURL(t *testing.T) {
	SetAppURL("")
	t.Cleanup(func() { SetAppURL(defaultAppURL) })
	got := TokenLink("/dogrula", "tok/en")
	if !strings.HasPrefix(got, defaultAppURL) {
		t.Fatalf("varsayılan taban yok: %q", got)
	}
	if !strings.Contains(got, "token=tok%2Fen") {
		t.Fatalf("token kaçışı yok: %q", got)
	}
}

func TestAppURLFromEnv(t *testing.T) {
	t.Setenv("APP_URL", "https://prod.ornek.test/")
	if got := AppURLFromEnv(); got != "https://prod.ornek.test" {
		t.Fatalf("got %q", got)
	}
}
