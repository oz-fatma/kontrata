package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

// TestAuthFlow Aşama 3 kimlik akışlarını tek oturumda uçtan uca doğrular.
func TestAuthFlow(t *testing.T) {
	dbName := fmt.Sprintf("%s_akis_%s", mongo.TestDatabasePrefix, bson.NewObjectID().Hex())
	t.Setenv("MONGO_DATABASE", dbName)
	ctx, env := setupRegisterRequired(t)
	t.Cleanup(func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := env.db.DropDatabase(dropCtx, dbName); err != nil {
			t.Errorf("akis veritabanı silinemedi: %v", err)
		}
	})
	requireReplicaSet(t, env.db, ctx)

	eposta := uniqueEmail()
	const cihazA, cihazB = "akis-cihaz-a", "akis-cihaz-b"

	t.Log("1. kayitOl → doğrulama kodu → epostaDogrula")
	kayit := postRegister(t, env.c, eposta, testPassword)
	if !kayit.KayitOl.Basarili {
		t.Fatal("kayıt başarısız")
	}
	dogrulama := tokenFromBody(env.mail.lastBody())
	if dogrulama == "" {
		t.Fatal("doğrulama kodu yok")
	}
	t.Logf("doğrulama kodu alındı uzunluk=%d", len(dogrulama))
	if !postVerify(t, env.c, dogrulama) {
		t.Fatal("e-posta doğrulanamadı")
	}
	user, err := env.users.GetByEmail(ctx, eposta)
	if err != nil || !user.EmailVerified {
		t.Fatal("kullanıcı doğrulanmamış")
	}

	t.Log("2. girisYap → MFA kodu → mfaDogrula")
	access, refresh := loginSessionDevice(t, env, eposta, testPassword, cihazA)
	if access == "" || refresh == "" {
		t.Fatal("oturum jetonları boş")
	}
	t.Log("erişim ve yenileme jetonu alındı")

	t.Log("3. jetonsuz korumalı sorgu reddedilir, jetonlu çalışır")
	var sozlesmeler struct{ Sozlesmeler []any }
	err = env.c.Post(`query { sozlesmeler { id } }`, &sozlesmeler)
	if err == nil || !strings.Contains(err.Error(), "kimlik doğrulaması gerekli") {
		t.Fatalf("jetonsuz sorgu: %v", err)
	}
	env.withDevice(access, cihazA).MustPost(`query { sozlesmeler { id } }`, &sozlesmeler)
	t.Log("jetonlu sozlesmeler kabul edildi")

	t.Log("4. jetonYenile: yeni çift, eski yenileme reddedilir")
	var yenile struct {
		JetonYenile struct {
			ErisimJetonu   string
			YenilemeJetonu string
		}
	}
	env.c.MustPost(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu yenilemeJetonu }
	}`, &yenile, client.Var("r", refresh))
	yeniAccess := yenile.JetonYenile.ErisimJetonu
	yeniRefresh := yenile.JetonYenile.YenilemeJetonu
	if yeniAccess == "" || yeniRefresh == "" || yeniRefresh == refresh {
		t.Fatal("yeni yenileme jetonu yok")
	}
	eskiYenileme := env.c.Post(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
	}`, &yenile, client.Var("r", refresh))
	assertYenilemeReddi(t, eskiYenileme)
	access, refresh = yeniAccess, yeniRefresh
	t.Log("eski yenileme jetonu reddedildi")

	t.Log("5. cihazlarim tek cihaz; aynı cihazdan ikinci giriş yeni kayıt açmaz")
	cA := env.withDevice(access, cihazA)
	var cihazlar struct {
		Cihazlarim []struct{ ID, Ad string }
	}
	cA.MustPost(`query { cihazlarim { id ad } }`, &cihazlar)
	if len(cihazlar.Cihazlarim) != 1 {
		t.Fatalf("cihaz sayısı = %d", len(cihazlar.Cihazlarim))
	}
	idA := cihazlar.Cihazlarim[0].ID
	loginSessionDevice(t, env, eposta, testPassword, cihazA)
	cA.MustPost(`query { cihazlarim { id ad } }`, &cihazlar)
	if len(cihazlar.Cihazlarim) != 1 || cihazlar.Cihazlarim[0].ID != idA {
		t.Fatal("ikinci girişte yeni cihaz oluştu")
	}
	t.Logf("cihaz %s tek kaldı", idA)

	t.Log("6. cihazKaldir oturumları iptal eder")
	accessB, refreshB := loginSessionDevice(t, env, eposta, testPassword, cihazB)
	cB := env.withDevice(accessB, cihazB)
	var silCihaz struct{ CihazKaldir bool }
	cB.MustPost(`mutation ($id: ID!) { cihazKaldir(id: $id) }`, &silCihaz, client.Var("id", idA))
	if !silCihaz.CihazKaldir {
		t.Fatal("cihazKaldir false")
	}
	err = env.c.Post(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
	}`, &yenile, client.Var("r", refresh))
	if err == nil {
		t.Fatal("kaldırılan cihazın oturumu hâlâ yenileniyor")
	}
	cB.MustPost(`query { cihazlarim { id } }`, &cihazlar)
	if len(cihazlar.Cihazlarim) != 1 || cihazlar.Cihazlarim[0].ID == idA {
		t.Fatal("yanlış cihaz kaldı")
	}
	t.Log("cihaz A oturumları iptal, cihaz B duruyor")

	t.Log("7. tumOturumlariKapat mevcut oturum hariç kapatır")
	_, extra1 := loginSessionDevice(t, env, eposta, testPassword, cihazB)
	_, extra2 := loginSessionDevice(t, env, eposta, testPassword, cihazB)
	cB = env.withDevice(accessB, cihazB)
	var kapat struct{ TumOturumlariKapat int32 }
	cB.MustPost(`mutation { tumOturumlariKapat }`, &kapat)
	if kapat.TumOturumlariKapat < 2 {
		t.Fatalf("iptal edilen = %d", kapat.TumOturumlariKapat)
	}
	for _, eskiYenileme := range []string{extra1, extra2} {
		err = env.c.Post(`mutation ($r: String!) {
			jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
		}`, &yenile, client.Var("r", eskiYenileme))
		assertYenilemeReddi(t, err)
	}
	var oturumlar struct {
		Oturumlarim []struct{ MevcutMu bool }
	}
	cB.MustPost(`query { oturumlarim { mevcutMu } }`, &oturumlar)
	if len(oturumlar.Oturumlarim) != 1 || !oturumlar.Oturumlarim[0].MevcutMu {
		t.Fatal("mevcut oturum kapanmış")
	}
	env.c.MustPost(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu yenilemeJetonu }
	}`, &yenile, client.Var("r", refreshB))
	if yenile.JetonYenile.ErisimJetonu == "" {
		t.Fatal("mevcut oturum yenilenemedi")
	}
	cB = env.withDevice(yenile.JetonYenile.ErisimJetonu, cihazB)
	t.Logf("iptal edilen oturum = %d, mevcut oturum yenilendi", kapat.TumOturumlariKapat)

	t.Log("8. verilerimiIndir hash ve token içermez")
	cB.MustPost(`mutation ($g: SozlesmeGirdi!) { sozlesmeOlustur(girdi: $g) { id } }`,
		new(struct{ SozlesmeOlustur struct{ ID string } }), client.Var("g", sozlesmeGirdi()))
	var indir struct{ VerilerimiIndir string }
	cB.MustPost(`query { verilerimiIndir }`, &indir)
	if indir.VerilerimiIndir == "" {
		t.Fatal("boş dışa aktarım")
	}
	yasak := []string{"sifreHash", "yenilemeTokenHash", "cihazParmakIzi", "kodHash"}
	for _, k := range yasak {
		if strings.Contains(indir.VerilerimiIndir, k) {
			t.Fatalf("dışa aktarımda %s var", k)
		}
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(indir.VerilerimiIndir), &parsed); err != nil {
		t.Fatal("JSON değil")
	}
	if parsed["eposta"] != eposta {
		t.Fatalf("eposta = %v", parsed["eposta"])
	}
	t.Log("dışa aktarım hash içermiyor")

	t.Log("9. hesapSilmeIste → hesapSil; kayıtlar silinir, denetim anonimleşir")
	var iste struct{ HesapSilmeIste bool }
	cB.MustPost(`mutation { hesapSilmeIste }`, &iste)
	if !iste.HesapSilmeIste {
		t.Fatal("hesapSilmeIste false")
	}
	silmeKodu := tokenAfterLabel(env.mail.lastBody(), "Hesap silme onay kodunuz:")
	if silmeKodu == "" {
		t.Fatal("silme kodu yok")
	}
	t.Logf("silme onay kodu alındı uzunluk=%d", len(silmeKodu))
	var sil struct{ HesapSil bool }
	env.c.MustPost(`mutation ($t: String!) { hesapSil(token: $t) }`, &sil, client.Var("t", silmeKodu))
	if !sil.HesapSil {
		t.Fatal("hesapSil false")
	}
	if _, err := env.users.GetByEmail(ctx, eposta); err == nil {
		t.Fatal("kullanıcı duruyor")
	}
	if n, _ := env.soz.CountByUser(ctx, user.ID); n != 0 {
		t.Fatalf("sözleşme kaldı %d", n)
	}
	if n, _ := env.tokens.CountByUser(ctx, user.ID); n != 0 {
		t.Fatalf("token kaldı %d", n)
	}
	if n, _ := env.mfa.CountByUser(ctx, user.ID); n != 0 {
		t.Fatalf("mfa kaldı %d", n)
	}
	if n, _ := env.sessions.CountByUser(ctx, user.ID); n != 0 {
		t.Fatalf("oturum kaldı %d", n)
	}
	if n, _ := env.devices.CountByUser(ctx, user.ID); n != 0 {
		t.Fatalf("cihaz kaldı %d", n)
	}
	if n, _ := env.audit.CountByUser(ctx, user.ID); n != 0 {
		t.Fatalf("denetim hâlâ kullanıcıya bağlı %d", n)
	}
	kayitDenetim, err := env.audit.Latest(ctx, repository.EventAccountDeleted)
	if err != nil {
		t.Fatal("HESAP_SILINDI yok")
	}
	silinmis, ok := kayitDenetim.UserID.(string)
	if !ok || silinmis != repository.UserDeleted {
		t.Fatalf("kullaniciId = %#v", kayitDenetim.UserID)
	}
	if kayitDenetim.IPAddress != "" || kayitDenetim.UserAgent != "" {
		t.Fatal("silme kaydında IP/UA duruyor")
	}
	t.Log("hesap silindi, denetim anonim")
}
