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
	"github.com/oz-fatma/kontrata/backend/internal/filestore"
	"github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
	"github.com/oz-fatma/kontrata/backend/internal/service"
)

type stubMailer struct {
	mu   sync.Mutex
	sent []struct{ to, subject, body string }
}

func (m *stubMailer) Send(to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, struct{ to, subject, body string }{to, subject, body})
	return nil
}

func (m *stubMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

func (m *stubMailer) countSubject(subject string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, s := range m.sent {
		if s.subject == subject {
			n++
		}
	}
	return n
}

func (m *stubMailer) lastBody() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		return ""
	}
	return m.sent[len(m.sent)-1].body
}

func tokenFromBody(body string) string {
	return tokenAfterLabel(body, "Doğrulama kodunuz:")
}

func tokenAfterLabel(body, label string) string {
	_, rest, ok := strings.Cut(body, label)
	if !ok {
		return ""
	}
	rest = strings.TrimSpace(rest)
	line, _, _ := strings.Cut(rest, "\n")
	return strings.TrimSpace(line)
}

const (
	testPassword  = "oniki-karakter"
	testJWTSecret = "test-jwt-secret-at-least-32-bytes!!"
	testUserAgent = "kontrata-test"
	testForwarded = "203.0.113.9, 10.0.0.1"
)

type registerEnv struct {
	h        http.Handler
	c        *client.Client
	users    *repository.UserRepository
	tokens   *repository.VerificationTokenRepository
	mfa      *repository.MFACodeRepository
	sessions *repository.SessionRepository
	soz      *repository.ContractRepository
	audit    *repository.AuditRepository
	mail     *stubMailer
	devices  *repository.DeviceRepository
	orgs     *repository.OrganizationRepository
	db       *mongo.Client
	sozSvc   *service.ContractService
	files    *filestore.Store
	prompts  *repository.PromptVersionRepository
	settings *repository.OrgSettingsRepository
	llmCalls *repository.LLMCallRepository
	davets   *repository.InviteRepository
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

func setupRegisterRequired(t *testing.T) (context.Context, registerEnv) {
	t.Helper()
	requireMongoURI(t)
	return setupRegister(t)
}

func setupRegister(t *testing.T) (context.Context, registerEnv) {
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

	users := repository.NewUserRepository(db)
	tokens := repository.NewVerificationTokenRepository(db)
	mfa := repository.NewMFACodeRepository(db)
	sessions := repository.NewSessionRepository(db)
	devices := repository.NewDeviceRepository(db)
	denetim := repository.NewAuditRepository(db)
	sozRepo := repository.NewContractRepository(db)
	orgs := repository.NewOrganizationRepository(db)
	davets := repository.NewInviteRepository(db)
	promptlar := repository.NewPromptVersionRepository(db)
	ayarlarRepo := repository.NewOrgSettingsRepository(db)
	llmCagrilari := repository.NewLLMCallRepository(db)
	for _, ensure := range []func(context.Context) error{
		users.EnsureIndexes, tokens.EnsureIndexes, mfa.EnsureIndexes,
		sessions.EnsureIndexes, devices.EnsureIndexes, denetim.EnsureIndexes, sozRepo.EnsureIndexes,
		orgs.EnsureIndexes, davets.EnsureIndexes, promptlar.EnsureIndexes, ayarlarRepo.EnsureIndexes,
		llmCagrilari.EnsureIndexes,
		users.BackfillAccountFields,
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
	authSvc.AttachOrgLLM(promptlar, ayarlarRepo)
	authSvc.AttachLLMCalls(llmCagrilari)
	sozSvc := service.NewContractService(sozRepo, users)
	sozSvc.AttachAudit(denetim)
	sozSvc.AttachOrgLLM(promptlar, ayarlarRepo)
	files, err := filestore.New(t.TempDir())
	if err != nil {
		t.Fatalf("dosya deposu: %v", err)
	}
	sozSvc.AttachExtract(files, nil, "")
	authSvc.AttachFiles(files)
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers:  &Resolver{Service: sozSvc, Auth: authSvc},
		Directives: DirectiveRoot{Auth: AuthDirective},
	}))
	srv.AddTransport(transport.POST{})
	gql := auth.RequestMiddleware(authSvc.BearerMiddleware(srv))
	mux := http.NewServeMux()
	mux.Handle("GET /dosya/{id}", auth.RequestMiddleware(authSvc.BearerMiddleware(http.HandlerFunc(sozSvc.ServeFile))))
	mux.Handle("/", gql)
	return ctx, registerEnv{
		h: mux, c: graphqlClient(mux, "", ""), users: users, tokens: tokens,
		mfa: mfa, sessions: sessions, soz: sozRepo, audit: denetim, mail: mail,
		devices: devices, orgs: orgs, db: db, sozSvc: sozSvc, files: files,
		prompts: promptlar, settings: ayarlarRepo, llmCalls: llmCagrilari, davets: davets,
	}
}

func (e registerEnv) withToken(access string) *client.Client {
	return graphqlClient(e.h, access, "")
}

func (e registerEnv) withDevice(access, deviceID string) *client.Client {
	return graphqlClient(e.h, access, deviceID)
}

func uniqueEmail() string {
	return "kayit-" + bson.NewObjectID().Hex() + "@ornek.test"
}

type registerResult struct {
	KayitOl struct {
		Basarili bool
		Mesaj    string
	}
}

func postRegister(t *testing.T, c *client.Client, eposta, sifre string) registerResult {
	t.Helper()
	var out registerResult
	c.MustPost(`mutation ($e: String!, $s: String!) {
		kayitOl(eposta: $e, sifre: $s) { basarili mesaj }
	}`, &out, client.Var("e", eposta), client.Var("s", sifre))
	return out
}

func postVerify(t *testing.T, c *client.Client, token string) bool {
	t.Helper()
	var out struct{ EpostaDogrula bool }
	c.MustPost(`mutation ($t: String!) { epostaDogrula(token: $t) }`, &out, client.Var("t", token))
	return out.EpostaDogrula
}

func registerVerified(t *testing.T, env registerEnv, eposta, sifre string) {
	t.Helper()
	postRegister(t, env.c, eposta, sifre)
	plain := tokenFromBody(env.mail.lastBody())
	if plain == "" {
		t.Fatal("doğrulama kodu yok")
	}
	if !postVerify(t, env.c, plain) {
		t.Fatal("e-posta doğrulanamadı")
	}
}

func loginSession(t *testing.T, env registerEnv, eposta, sifre string) (access, refresh string) {
	t.Helper()
	return loginOn(t, env, env.c, eposta, sifre)
}

func loginSessionDevice(t *testing.T, env registerEnv, eposta, sifre, deviceID string) (access, refresh string) {
	t.Helper()
	return loginOn(t, env, env.withDevice("", deviceID), eposta, sifre)
}

func loginOn(t *testing.T, env registerEnv, c *client.Client, eposta, sifre string) (access, refresh string) {
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
	kod := tokenAfterLabel(env.mail.lastBody(), "Giriş kodunuz:")
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

func TestRegisterAndVerifyEmail(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	first := postRegister(t, env.c, eposta, testPassword)
	if !first.KayitOl.Basarili || first.KayitOl.Mesaj == "" {
		t.Fatal("kayıt yanıtı beklenen değil")
	}
	plain := tokenFromBody(env.mail.lastBody())
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
	if stored.Used {
		t.Fatal("yeni token kullanılmış görünüyor")
	}
	kayitDenetim, err := env.audit.Latest(ctx, repository.EventRegister)
	if err != nil {
		t.Fatalf("denetim okunamadı")
	}
	if kayitDenetim.IPAddress != "203.0.113.9" {
		t.Fatalf("ipAdresi = %q", kayitDenetim.IPAddress)
	}
	if kayitDenetim.UserAgent != testUserAgent {
		t.Fatalf("kullaniciAjani = %q", kayitDenetim.UserAgent)
	}

	if !postVerify(t, env.c, plain) {
		t.Fatal("geçerli kod reddedildi")
	}
	user, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if !user.EmailVerified || user.Status != repository.StatusActive {
		t.Fatalf("doğrulama sonrası durum = %s dogrulandi=%v", user.Status, user.EmailVerified)
	}
	if user.AccountType != repository.AccountIndividual || user.Role != repository.RoleOwner {
		t.Fatalf("hesapTipi=%s rol=%s", user.AccountType, user.Role)
	}
	if !user.OrganizationID.IsZero() {
		t.Fatal("bireysel hesapta organizasyonId dolu")
	}
	if postVerify(t, env.c, plain) {
		t.Fatal("kullanılmış kod ikinci kez kabul edildi")
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	postRegister(t, env.c, eposta, testPassword)
	plain := tokenFromBody(env.mail.lastBody())
	if plain == "" {
		t.Fatal("doğrulama kodu yok")
	}
	doc, err := env.tokens.GetByHash(ctx, auth.HashToken(plain))
	if err != nil {
		t.Fatalf("token okunamadı")
	}
	doc.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	if err := env.tokens.Update(ctx, doc); err != nil {
		t.Fatalf("token güncellenemedi")
	}
	if postVerify(t, env.c, plain) {
		t.Fatal("süresi geçmiş kod kabul edildi")
	}
	user, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if user.EmailVerified {
		t.Fatal("süresi geçmiş kod hesabı doğruladı")
	}
}

func TestDuplicateEmailLeavesOriginal(t *testing.T) {
	ctx, env := setupRegister(t)
	eposta := uniqueEmail()
	first := postRegister(t, env.c, eposta, testPassword)
	plain := tokenFromBody(env.mail.lastBody())
	user, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	hash := user.PasswordHash
	mails := env.mail.count()

	second := postRegister(t, env.c, eposta, "baska-sifre-12")
	if second.KayitOl.Basarili != first.KayitOl.Basarili || second.KayitOl.Mesaj != first.KayitOl.Mesaj {
		t.Fatal("ikinci kayıt farklı yanıt verdi")
	}
	if env.mail.count() != mails {
		t.Fatal("ikinci kayıt yeni e-posta gönderdi")
	}
	again, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	if again.PasswordHash != hash {
		t.Fatal("ikinci kayıt şifre özetini değiştirdi")
	}
	if again.Status != repository.StatusPending {
		t.Fatalf("durum = %s", again.Status)
	}
	if !postVerify(t, env.c, plain) {
		t.Fatal("ilk kod ikinci kayıttan sonra geçersiz")
	}
}
