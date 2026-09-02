package graph

import (
	"context"
	"net/http"
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

func (m *stubMailer) countKonu(konu string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, s := range m.sent {
		if s.konu == konu {
			n++
		}
	}
	return n
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

const (
	testSifre     = "oniki-karakter"
	testJWTSecret = "test-jwt-secret-at-least-32-bytes!!"
	testUserAgent = "kontrata-test"
	testForwarded = "203.0.113.9, 10.0.0.1"
)

type kayitEnv struct {
	h        http.Handler
	c        *client.Client
	users    *repository.KullaniciRepository
	tokens   *repository.DogrulamaTokenRepository
	mfa      *repository.MFAKoduRepository
	sessions *repository.OturumRepository
	soz      *repository.SozlesmeRepository
	audit    *repository.DenetimRepository
	mail     *stubMailer
	devices  *repository.CihazRepository
	orgs     *repository.OrganizasyonRepository
	db       *mongo.Client
}

func testParams() auth.Params {
	return auth.Params{Time: 1, Memory: 8 * 1024, Threads: 1, KeyLen: 32, SaltLen: 16}
}

func graphqlClient(h http.Handler, access, deviceID string) *client.Client {
	opts := []client.Option{
		client.AddHeader("User-Agent", testUserAgent),
		client.AddHeader("X-Forwarded-For", testForwarded),
		client.AddHeader("Accept-Language", "tr-TR"),
	}
	if access != "" {
		opts = append(opts, client.AddHeader("Authorization", "Bearer "+access))
	}
	if deviceID != "" {
		opts = append(opts, client.AddHeader("X-Device-Id", deviceID))
	}
	return client.New(h, opts...)
}

func requireMongoURI(t *testing.T) {
	t.Helper()
	if strings.TrimSpace(os.Getenv("MONGO_URI")) == "" {
		t.Fatal("MONGO_URI yok; akış testi atlanamaz. docker compose up -d çalıştırın, backend/.env dosyasına MONGO_URI=mongodb://localhost:27017 yazın (veya değişkeni export edin), ardından make test-akis / make test-org.")
	}
}

func requireReplicaSet(t *testing.T, db *mongo.Client, ctx context.Context) {
	t.Helper()
	if !db.ReplicaSet(ctx) {
		t.Fatal("replica set gerekli; akış testi atlanamaz. docker compose up -d çalıştırın (mongo:8 --replSet rs0).")
	}
}

func setupKayitRequired(t *testing.T) (context.Context, kayitEnv) {
	t.Helper()
	requireMongoURI(t)
	return setupKayit(t)
}

func setupKayit(t *testing.T) (context.Context, kayitEnv) {
	t.Helper()
	uri := strings.TrimSpace(os.Getenv("MONGO_URI"))
	if uri == "" {
		t.Skip("MONGO_URI yok")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
	mfa := repository.NewMFAKoduRepository(db)
	sessions := repository.NewOturumRepository(db)
	devices := repository.NewCihazRepository(db)
	denetim := repository.NewDenetimRepository(db)
	sozRepo := repository.NewSozlesmeRepository(db)
	orgs := repository.NewOrganizasyonRepository(db)
	davets := repository.NewDavetRepository(db)
	for _, ensure := range []func(context.Context) error{
		users.EnsureIndexes, tokens.EnsureIndexes, mfa.EnsureIndexes,
		sessions.EnsureIndexes, devices.EnsureIndexes, denetim.EnsureIndexes, sozRepo.EnsureIndexes,
		orgs.EnsureIndexes, davets.EnsureIndexes, users.BackfillHesapAlanlari,
	} {
		if err := ensure(ctx); err != nil {
			t.Fatalf("indeks oluşturulamadı")
		}
	}
	mail := &stubMailer{}
	signer, err := auth.NewJWT([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	authSvc := service.NewAuthService(users, tokens, mfa, sessions, devices, sozRepo, orgs, davets, denetim, mail, testParams(), signer, db)
	sozSvc := service.NewSozlesmeService(sozRepo, users)
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers:  &Resolver{Service: sozSvc, Auth: authSvc},
		Directives: DirectiveRoot{Auth: AuthDirective},
	}))
	srv.AddTransport(transport.POST{})
	h := auth.RequestMiddleware(authSvc.BearerMiddleware(srv))
	return ctx, kayitEnv{
		h: h, c: graphqlClient(h, "", ""), users: users, tokens: tokens,
		mfa: mfa, sessions: sessions, soz: sozRepo, audit: denetim, mail: mail,
		devices: devices, orgs: orgs, db: db,
	}
}

func (e kayitEnv) withToken(access string) *client.Client {
	return graphqlClient(e.h, access, "")
}

func (e kayitEnv) withDevice(access, deviceID string) *client.Client {
	return graphqlClient(e.h, access, deviceID)
}

func uniqueEposta() string {
	return "kayit-" + bson.NewObjectID().Hex() + "@ornek.test"
}

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

func registerVerified(t *testing.T, env kayitEnv, eposta, sifre string) {
	t.Helper()
	postKayit(t, env.c, eposta, sifre)
	plain := tokenFromGovde(env.mail.lastGovde())
	if plain == "" {
		t.Fatal("doğrulama kodu yok")
	}
	if !postDogrula(t, env.c, plain) {
		t.Fatal("e-posta doğrulanamadı")
	}
}

func loginSession(t *testing.T, env kayitEnv, eposta, sifre string) (access, refresh string) {
	t.Helper()
	return loginOn(t, env, env.c, eposta, sifre)
}

func loginSessionDevice(t *testing.T, env kayitEnv, eposta, sifre, deviceID string) (access, refresh string) {
	t.Helper()
	return loginOn(t, env, env.withDevice("", deviceID), eposta, sifre)
}

func loginOn(t *testing.T, env kayitEnv, c *client.Client, eposta, sifre string) (access, refresh string) {
	t.Helper()
	var giris struct {
		GirisYap struct {
			MfaGerekli  bool
			GeciciToken string
		}
	}
	env.c.MustPost(`mutation ($e: String!, $s: String!) {
		girisYap(eposta: $e, sifre: $s) { mfaGerekli geciciToken }
	}`, &giris, client.Var("e", eposta), client.Var("s", sifre))
	if !giris.GirisYap.MfaGerekli || giris.GirisYap.GeciciToken == "" {
		t.Fatal("MFA bekleniyordu")
	}
	kod := tokenAfterLabel(env.mail.lastGovde(), "Giriş kodunuz:")
	if kod == "" {
		t.Fatal("MFA kodu yok")
	}
	var oturum struct {
		MfaDogrula struct {
			ErisimJetonu   string
			YenilemeJetonu string
		}
	}
	if err := c.Post(`mutation ($g: String!, $k: String!) {
		mfaDogrula(geciciToken: $g, kod: $k) { erisimJetonu yenilemeJetonu }
	}`, &oturum, client.Var("g", giris.GirisYap.GeciciToken), client.Var("k", kod)); err != nil {
		t.Fatalf("mfaDogrula: %v", err)
	}
	if oturum.MfaDogrula.ErisimJetonu == "" || oturum.MfaDogrula.YenilemeJetonu == "" {
		t.Fatal("oturum jetonları boş")
	}
	return oturum.MfaDogrula.ErisimJetonu, oturum.MfaDogrula.YenilemeJetonu
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
	if kayitDenetim.KullaniciAjani != testUserAgent {
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
	if user.HesapTipi != repository.HesapBireysel || user.Rol != repository.RolSahip {
		t.Fatalf("hesapTipi=%s rol=%s", user.HesapTipi, user.Rol)
	}
	if !user.OrganizasyonID.IsZero() {
		t.Fatal("bireysel hesapta organizasyonId dolu")
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
