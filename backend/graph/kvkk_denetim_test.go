package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/llm"
	"github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

func TestKVKKSilmeZinciri(t *testing.T) {
	dbName := fmt.Sprintf("%s_kvkk_%s", mongo.TestDatabasePrefix, bson.NewObjectID().Hex())
	t.Setenv("MONGO_DATABASE", dbName)
	ctx, env := setupRegisterRequired(t)
	t.Cleanup(func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := env.db.DropDatabase(dropCtx, dbName); err != nil {
			t.Errorf("veritabanı silinemedi: %v", err)
		}
	})
	requireReplicaSet(t, env.db, ctx)

	eposta := uniqueEmail()
	davetEposta := uniqueEmail()
	const cihaz = "kvkk-silme-cihaz"
	postRegisterCorporate(t, env.c, eposta, testPassword, "KVKK Otel")
	if !postVerify(t, env.c, tokenFromBody(env.mail.lastBody())) {
		t.Fatal("e-posta doğrulanamadı")
	}
	sahip, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatal(err)
	}
	orgID := sahip.OrganizationID
	access, _ := loginSessionDevice(t, env, eposta, testPassword, cihaz)
	c := env.withDevice(access, cihaz)

	pdfA := testdataPDF(t, "argos-megep.pdf")
	pdfB := testdataPDF(t, "tui-2026-yaz.pdf")
	a := uploadPDF(t, ctx, env, sahip.ID, "argos.pdf", pdfA)
	b := uploadPDF(t, ctx, env, sahip.ID, "tui.pdf", pdfB)
	fileA := filepath.Join(env.files.Dir(), a.StoredFileID)
	fileB := filepath.Join(env.files.Dir(), b.StoredFileID)
	if _, err := os.Stat(fileA); err != nil {
		t.Fatalf("yüklenen PDF yok: %v", err)
	}
	if _, err := os.Stat(fileB); err != nil {
		t.Fatalf("yüklenen PDF yok: %v", err)
	}

	c.MustPost(`mutation ($id: ID!, $a: String!, $d: JSON!) {
		sozlesmeAlanGuncelle(id: $id, alanYolu: $a, deger: $d) { id }
	}`, new(struct{ SozlesmeAlanGuncelle struct{ ID string } }),
		client.Var("id", a.ID.Hex()), client.Var("a", "meta.otelAdi"), client.Var("d", "KVKK Otel"))

	c.MustPost(`mutation ($id: ID!, $g: SozlesmeGirdi!) {
		sozlesmeGuncelle(id: $id, girdi: $g) { id durum }
	}`, new(struct{ SozlesmeGuncelle struct{ ID, Durum string } }),
		client.Var("id", b.ID.Hex()),
		client.Var("g", map[string]any{"durum": "INCELENMEYI_BEKLIYOR"}))
	c.MustPost(`mutation ($id: ID!) { sozlesmeOnayla(id: $id) { id durum } }`,
		new(struct{ SozlesmeOnayla struct{ ID, Durum string } }), client.Var("id", b.ID.Hex()))

	c.MustPost(`mutation ($e: String!, $r: Rol!) { uyeDavetEt(eposta: $e, rol: $r) }`,
		new(struct{ UyeDavetEt bool }), client.Var("e", davetEposta), client.Var("r", "YONETICI"))

	c.MustPost(`query { ayarlar { maxToken } }`, new(struct{ Ayarlar struct{ MaxToken int32 } }))
	c.MustPost(`mutation ($t: PromptTipi!, $i: String!) { promptGuncelle(tip: $t, icerik: $i) { id } }`,
		new(struct{ PromptGuncelle struct{ ID string } }),
		client.Var("t", "OKUYUCU"), client.Var("i", "KVKK denetim prompt metni, maskeleme kapatilamaz."))

	now := time.Now().UTC()
	if err := env.llmCalls.Insert(ctx, &repository.LLMCall{
		OrgID:      orgID,
		ContractID: a.ID,
		Agent:      llm.AgentReader,
		Endpoint:   llm.EndpointUC1,
		Start:      now,
		End:        now,
		DurationMs: 10,
		Success:    true,
		ErrorType:  llm.HataYok,
		Attempt:    1,
	}); err != nil {
		t.Fatalf("llm kaydı: %v", err)
	}

	var iste struct{ HesapSilmeIste bool }
	c.MustPost(`mutation { hesapSilmeIste }`, &iste)
	if !iste.HesapSilmeIste {
		t.Fatal("hesapSilmeIste false")
	}
	kod := tokenAfterLabel(env.mail.lastBody(), "Hesap silme onay kodunuz:")
	if kod == "" {
		t.Fatal("silme kodu yok")
	}
	var sil struct{ HesapSil bool }
	env.c.MustPost(`mutation ($t: String!) { hesapSil(token: $t) }`, &sil, client.Var("t", kod))
	if !sil.HesapSil {
		t.Fatal("hesapSil false")
	}

	if _, err := env.users.GetByEmail(ctx, eposta); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("kullanıcı duruyor: %v", err)
	}
	if n, _ := env.users.CountByOrg(ctx, orgID); n != 0 {
		t.Fatalf("kullanicilar org kaydı kaldı n=%d", n)
	}
	if n, _ := env.sessions.CountByUser(ctx, sahip.ID); n != 0 {
		t.Fatalf("oturumlar kaldı n=%d", n)
	}
	if n, _ := env.devices.CountByUser(ctx, sahip.ID); n != 0 {
		t.Fatalf("cihazlar kaldı n=%d", n)
	}
	if n, _ := env.tokens.CountByUser(ctx, sahip.ID); n != 0 {
		t.Fatalf("dogrulama_tokenlari kaldı n=%d", n)
	}
	if n, _ := env.mfa.CountByUser(ctx, sahip.ID); n != 0 {
		t.Fatalf("mfa_kodlari kaldı n=%d", n)
	}
	if n, _ := env.soz.CountByOrg(ctx, orgID); n != 0 {
		t.Fatalf("sozlesmeler kaldı n=%d", n)
	}
	if n, _ := env.prompts.CountByOrg(ctx, orgID); n != 0 {
		t.Fatalf("prompt_surumleri kaldı n=%d", n)
	}
	if n, _ := env.settings.CountByOrg(ctx, orgID); n != 0 {
		t.Fatalf("ayarlar kaldı n=%d", n)
	}
	if n, _ := env.llmCalls.CountByOrg(ctx, orgID); n != 0 {
		t.Fatalf("llm_cagrilari kaldı n=%d", n)
	}
	if _, err := os.Stat(fileA); !os.IsNotExist(err) {
		t.Fatalf("PDF duruyor: %s", fileA)
	}
	if _, err := os.Stat(fileB); !os.IsNotExist(err) {
		t.Fatalf("PDF duruyor: %s", fileB)
	}
	if n, _ := env.audit.CountByUser(ctx, sahip.ID); n != 0 {
		t.Fatalf("denetim hâlâ kullanıcıya bağlı n=%d", n)
	}
	kayitlar, err := env.audit.ListDeleted(ctx)
	if err != nil || len(kayitlar) == 0 {
		t.Fatalf("anonim denetim yok: n=%d err=%v", len(kayitlar), err)
	}
	for i := range kayitlar {
		id, ok := kayitlar[i].UserID.(string)
		if !ok || id != repository.UserDeleted {
			t.Fatalf("kullaniciId = %#v", kayitlar[i].UserID)
		}
		if kayitlar[i].IPAddress != "" || kayitlar[i].UserAgent != "" {
			t.Fatalf("IP/UA duruyor olay=%s", kayitlar[i].Event)
		}
	}
}

func TestKVKKVeriIhraci(t *testing.T) {
	dbName := fmt.Sprintf("%s_kvkkexp_%s", mongo.TestDatabasePrefix, bson.NewObjectID().Hex())
	t.Setenv("MONGO_DATABASE", dbName)
	ctx, env := setupRegisterRequired(t)
	t.Cleanup(func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := env.db.DropDatabase(dropCtx, dbName); err != nil {
			t.Errorf("veritabanı silinemedi: %v", err)
		}
	})
	requireReplicaSet(t, env.db, ctx)

	eposta := uniqueEmail()
	const cihaz = "kvkk-ihrac-cihaz"
	postRegisterCorporate(t, env.c, eposta, testPassword, "İhraç Otel")
	if !postVerify(t, env.c, tokenFromBody(env.mail.lastBody())) {
		t.Fatal("e-posta doğrulanamadı")
	}
	access, _ := loginSessionDevice(t, env, eposta, testPassword, cihaz)
	c := env.withDevice(access, cihaz)

	var indir struct{ VerilerimiIndir string }
	c.MustPost(`query { verilerimiIndir }`, &indir)
	if indir.VerilerimiIndir == "" {
		t.Fatal("boş dışa aktarım")
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(indir.VerilerimiIndir), &parsed); err != nil {
		t.Fatal("JSON değil")
	}
	if parsed["eposta"] != eposta {
		t.Fatalf("eposta = %v", parsed["eposta"])
	}
	assertExportHasNoSecrets(t, parsed, "")
}

func testdataPDF(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join("..", "..", "testdata", "sozlesmeler", name)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("testdata yok: %s", p)
	}
	return p
}

func uploadPDF(t *testing.T, ctx context.Context, env registerEnv, userID bson.ObjectID, name, path string) *repository.Contract {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	uctx := auth.WithIdentity(ctx, auth.Identity{UserID: userID})
	m, err := env.sozSvc.Upload(uctx, name, f)
	if err != nil {
		t.Fatalf("yükleme: %v", err)
	}
	doc, err := env.soz.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.StoredFileID == "" {
		t.Fatal("saklanan dosya kimliği yok")
	}
	return doc
}

func assertExportHasNoSecrets(t *testing.T, v any, path string) {
	t.Helper()
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			lk := strings.ToLower(k)
			switch {
			case strings.Contains(lk, "hash"),
				strings.Contains(lk, "token"),
				strings.Contains(lk, "sifre"),
				lk == "kod",
				strings.Contains(lk, "mfakod"),
				lk == "mfa":
				t.Fatalf("dışa aktarımda yasak alan %s", path+"."+k)
			}
			assertExportHasNoSecrets(t, val, path+"."+k)
		}
	case []any:
		for i, val := range x {
			assertExportHasNoSecrets(t, val, fmt.Sprintf("%s[%d]", path, i))
		}
	case string:
		if strings.HasPrefix(x, "$argon2") {
			t.Fatalf("dışa aktarımda şifre özeti %s", path)
		}
	}
}
