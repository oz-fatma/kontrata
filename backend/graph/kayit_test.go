package graph

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
	"github.com/oz-fatma/kontrata/backend/internal/service"
)

type stubMailer struct {
	mu   sync.Mutex
	sent []struct{ alici, konu, govde string }
}

func (m *stubMailer) Gonder(alici, konu, govde string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, struct{ alici, konu, govde string }{alici, konu, govde})
	return nil
}

func (m *stubMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

func (m *stubMailer) lastGovde() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		return ""
	}
	return m.sent[len(m.sent)-1].govde
}

func tokenFromGovde(govde string) string {
	return tokenAfterLabel(govde, "Doğrulama kodunuz:")
}

func tokenAfterLabel(govde, label string) string {
	_, rest, ok := strings.Cut(govde, label)
	if !ok {
		return ""
	}
	rest = strings.TrimSpace(rest)
	line, _, _ := strings.Cut(rest, "\n")
	return strings.TrimSpace(line)
}

type kayitEnv struct {
	c      *client.Client
	users  *repository.KullaniciRepository
	tokens *repository.DogrulamaTokenRepository
	audit  *repository.DenetimRepository
	mail   *stubMailer
}

func setupKayit(t *testing.T) (context.Context, kayitEnv) {
	t.Helper()
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		t.Skip("MONGO_URI yok")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	db, err := mongo.Connect(ctx, uri)
	if err != nil {
		t.Fatalf("mongo bağlanamadı")
	}
	t.Cleanup(func() {
		if err := db.Disconnect(context.Background()); err != nil {
			t.Errorf("veritabanı bağlantısı kapatılamadı: %v", err)
		}
	})

	users := repository.NewKullaniciRepository(db)
	tokens := repository.NewDogrulamaTokenRepository(db)
	denetim := repository.NewDenetimRepository(db)
	for _, ensure := range []func(context.Context) error{users.EnsureIndexes, tokens.EnsureIndexes, denetim.EnsureIndexes} {
		if err := ensure(ctx); err != nil {
			t.Fatalf("indeks oluşturulamadı")
		}
	}
	mail := &stubMailer{}
	params := auth.Params{Time: 1, Memory: 8 * 1024, Threads: 1, KeyLen: 32, SaltLen: 16}
	authSvc := service.NewAuthService(users, tokens, denetim, mail, params)
	srv := handler.New(NewExecutableSchema(Config{Resolvers: &Resolver{Auth: authSvc}}))
	srv.AddTransport(transport.POST{})
	h := auth.RequestMiddleware(srv)
	c := client.New(h,
		client.AddHeader("User-Agent", "kontrata-test"),
		client.AddHeader("X-Forwarded-For", "203.0.113.9, 10.0.0.1"),
	)
	return ctx, kayitEnv{c: c, users: users, tokens: tokens, audit: denetim, mail: mail}
}

func uniqueEposta() string {
	return "kayit-" + bson.NewObjectID().Hex() + "@ornek.test"
}

const testSifre = "oniki-karakter"

type kayitYaniti struct {
	KayitOl struct {
		Basarili bool
		Mesaj    string
	}
}

func postKayit(t *testing.T, c *client.Client, eposta, sifre string) kayitYaniti {
	t.Helper()
	var out kayitYaniti
	c.MustPost(`mutation ($e: String!, $s: String!) {
		kayitOl(eposta: $e, sifre: $s) { basarili mesaj }
	}`, &out, client.Var("e", eposta), client.Var("s", sifre))
	return out
}

func postDogrula(t *testing.T, c *client.Client, token string) bool {
	t.Helper()
	var out struct{ EpostaDogrula bool }
	c.MustPost(`mutation ($t: String!) { epostaDogrula(token: $t) }`, &out, client.Var("t", token))
	return out.EpostaDogrula
}

func TestKayitOlVeEpostaDogrula(t *testing.T) {
	ctx, env := setupKayit(t)
	eposta := uniqueEposta()
	first := postKayit(t, env.c, eposta, testSifre)
	if !first.KayitOl.Basarili || first.KayitOl.Mesaj == "" {
		t.Fatal("kayıt yanıtı beklenen değil")
	}
	plain := tokenFromGovde(env.mail.lastGovde())
	if plain == "" {
		t.Fatal("doğrulama kodu e-postada yok")
	}
	hash := auth.HashToken(plain)
	stored, err := env.tokens.GetByHash(ctx, hash)
	if err != nil {
		t.Fatalf("token okunamadı")
	}
	if stored.Token == plain {
		t.Fatal("düz metin token yazılmış")
	}
	if stored.Kullanildi {
		t.Fatal("yeni token kullanılmış görünüyor")
	}
	kayitDenetim, err := env.audit.Latest(ctx, repository.OlayKayit)
	if err != nil {
		t.Fatalf("denetim okunamadı")
	}
	if kayitDenetim.IPAdresi != "203.0.113.9" {
		t.Fatalf("ipAdresi = %q", kayitDenetim.IPAdresi)
	}
	if kayitDenetim.KullaniciAjani != "kontrata-test" {
		t.Fatalf("kullaniciAjani = %q", kayitDenetim.KullaniciAjani)
	}

	if !postDogrula(t, env.c, plain) {
		t.Fatal("geçerli kod reddedildi")
	}
	user, err := env.users.GetByEposta(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if !user.EpostaDogrulandi || user.Durum != repository.DurumAktif {
		t.Fatalf("doğrulama sonrası durum = %s dogrulandi=%v", user.Durum, user.EpostaDogrulandi)
	}
	if postDogrula(t, env.c, plain) {
		t.Fatal("kullanılmış kod ikinci kez kabul edildi")
	}
}

func TestSuresiGecmisTokenReddedilir(t *testing.T) {
	ctx, env := setupKayit(t)
	eposta := uniqueEposta()
	postKayit(t, env.c, eposta, testSifre)
	plain := tokenFromGovde(env.mail.lastGovde())
	if plain == "" {
		t.Fatal("doğrulama kodu yok")
	}
	doc, err := env.tokens.GetByHash(ctx, auth.HashToken(plain))
	if err != nil {
		t.Fatalf("token okunamadı")
	}
	doc.SonKullanma = time.Now().UTC().Add(-time.Minute)
	if err := env.tokens.Update(ctx, doc); err != nil {
		t.Fatalf("token güncellenemedi")
	}
	if postDogrula(t, env.c, plain) {
		t.Fatal("süresi geçmiş kod kabul edildi")
	}
	user, err := env.users.GetByEposta(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if user.EpostaDogrulandi {
		t.Fatal("süresi geçmiş kod hesabı doğruladı")
	}
}

func TestAyniEpostaIkinciKayitIlkiniBozmaz(t *testing.T) {
	ctx, env := setupKayit(t)
	eposta := uniqueEposta()
	first := postKayit(t, env.c, eposta, testSifre)
	plain := tokenFromGovde(env.mail.lastGovde())
	user, err := env.users.GetByEposta(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	hash := user.SifreHash
	mails := env.mail.count()

	second := postKayit(t, env.c, eposta, "baska-sifre-12")
	if second.KayitOl.Basarili != first.KayitOl.Basarili || second.KayitOl.Mesaj != first.KayitOl.Mesaj {
		t.Fatal("ikinci kayıt farklı yanıt verdi")
	}
	if env.mail.count() != mails {
		t.Fatal("ikinci kayıt yeni e-posta gönderdi")
	}
	again, err := env.users.GetByEposta(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if again.SifreHash != hash {
		t.Fatal("ikinci kayıt şifre özetini değiştirdi")
	}
	if again.Durum != repository.DurumBeklemede {
		t.Fatalf("durum = %s", again.Durum)
	}
	if !postDogrula(t, env.c, plain) {
		t.Fatal("ilk kod ikinci kayıttan sonra geçersiz")
	}
}
