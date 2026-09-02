package graph

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

// TestOrganizasyonAkis kurumsal kayıt, davet, yetki matrisi ve kiracı yalıtımını doğrular.
func TestOrganizasyonAkis(t *testing.T) {
	dbName := fmt.Sprintf("%s_org_%s", mongo.TestDatabasePrefix, bson.NewObjectID().Hex())
	t.Setenv("MONGO_DATABASE", dbName)
	ctx, env := setupKayitRequired(t)
	t.Cleanup(func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := env.db.DropDatabase(dropCtx, dbName); err != nil {
			t.Errorf("org veritabanı silinemedi: %v", err)
		}
	})
	requireReplicaSet(t, env.db, ctx)

	sahipEposta := uniqueEposta()
	yoneticiEposta := uniqueEposta()
	izleyiciEposta := uniqueEposta()
	digerSahipEposta := uniqueEposta()
	const cihaz = "org-akis-cihaz"
	const orgAd = "Akdeniz Otelcilik"

	t.Log("1. kurumsal kayitOl → organizasyon ve SAHIP")
	postKayitKurumsal(t, env.c, sahipEposta, testSifre, orgAd)
	dogrulama := tokenFromGovde(env.mail.lastGovde())
	if dogrulama == "" {
		t.Fatal("doğrulama kodu yok")
	}
	if !postDogrula(t, env.c, dogrulama) {
		t.Fatal("e-posta doğrulanamadı")
	}
	sahip, err := env.users.GetByEposta(ctx, sahipEposta)
	if err != nil {
		t.Fatalf("sahip okunamadı: %v", err)
	}
	if sahip.HesapTipi != repository.HesapKurumsal || sahip.Rol != repository.RolSahip {
		t.Fatalf("hesapTipi=%s rol=%s", sahip.HesapTipi, sahip.Rol)
	}
	if sahip.OrganizasyonID.IsZero() {
		t.Fatal("organizasyon bağlanmadı")
	}
	org, err := env.orgs.GetByID(ctx, sahip.OrganizasyonID)
	if err != nil {
		t.Fatalf("organizasyon okunamadı: %v", err)
	}
	if org.Ad != orgAd || org.SahipKullaniciID != sahip.ID || org.Durum != repository.OrgDurumAktif {
		t.Fatalf("organizasyon ad=%s durum=%s", org.Ad, org.Durum)
	}
	sahipAccess, _ := loginSessionDevice(t, env, sahipEposta, testSifre, cihaz)
	cSahip := env.withDevice(sahipAccess, cihaz)
	var orgum struct {
		Organizasyonum *struct{ ID, Ad, Durum string }
	}
	cSahip.MustPost(`query { organizasyonum { id ad durum } }`, &orgum)
	if orgum.Organizasyonum == nil || orgum.Organizasyonum.ID != org.ID.Hex() {
		t.Fatal("organizasyonum boş veya yanlış")
	}
	t.Logf("organizasyon %s, sahip SAHIP", org.ID.Hex())

	t.Log("2. üye davet → kabul → YONETICI")
	var davet struct{ UyeDavetEt bool }
	cSahip.MustPost(`mutation ($e: String!, $r: Rol!) { uyeDavetEt(eposta: $e, rol: $r) }`,
		&davet, client.Var("e", yoneticiEposta), client.Var("r", "YONETICI"))
	if !davet.UyeDavetEt {
		t.Fatal("uyeDavetEt false")
	}
	davetKodu := tokenAfterLabel(env.mail.lastGovde(), "Davet kodunuz:")
	if davetKodu == "" {
		t.Fatal("davet kodu yok")
	}
	var kabul struct{ DavetiKabulEt bool }
	env.c.MustPost(`mutation ($t: String!, $s: String!) { davetiKabulEt(token: $t, sifre: $s) }`,
		&kabul, client.Var("t", davetKodu), client.Var("s", testSifre))
	if !kabul.DavetiKabulEt {
		t.Fatal("davetiKabulEt false")
	}
	yonetici, err := env.users.GetByEposta(ctx, yoneticiEposta)
	if err != nil {
		t.Fatalf("yönetici okunamadı: %v", err)
	}
	if yonetici.Rol != repository.RolYonetici || yonetici.OrganizasyonID != org.ID {
		t.Fatalf("yönetici rol=%s org=%s", yonetici.Rol, yonetici.OrganizasyonID.Hex())
	}
	yoneticiAccess, _ := loginSessionDevice(t, env, yoneticiEposta, testSifre, cihaz)
	cYonetici := env.withDevice(yoneticiAccess, cihaz)
	var uyeler struct {
		Uyeler []struct{ ID, Rol string }
	}
	cYonetici.MustPost(`query { uyeler { id rol } }`, &uyeler)
	if len(uyeler.Uyeler) != 2 {
		t.Fatalf("üye sayısı = %d", len(uyeler.Uyeler))
	}
	t.Log("YONETICI daveti kabul etti ve üyeleri gördü")

	t.Log("3. GORUNTULEYICI sozlesmeSil doğrudan reddedilir")
	cSahip.MustPost(`mutation ($e: String!, $r: Rol!) { uyeDavetEt(eposta: $e, rol: $r) }`,
		&davet, client.Var("e", izleyiciEposta), client.Var("r", "GORUNTULEYICI"))
	izleyiciKodu := tokenAfterLabel(env.mail.lastGovde(), "Davet kodunuz:")
	if izleyiciKodu == "" {
		t.Fatal("izleyici davet kodu yok")
	}
	env.c.MustPost(`mutation ($t: String!, $s: String!) { davetiKabulEt(token: $t, sifre: $s) }`,
		&kabul, client.Var("t", izleyiciKodu), client.Var("s", testSifre))
	if !kabul.DavetiKabulEt {
		t.Fatal("izleyici daveti kabul edilmedi")
	}
	var olustur struct {
		SozlesmeOlustur struct{ ID string }
	}
	cSahip.MustPost(`mutation ($g: SozlesmeGirdi!) { sozlesmeOlustur(girdi: $g) { id } }`,
		&olustur, client.Var("g", sozlesmeGirdi()))
	sozID := olustur.SozlesmeOlustur.ID
	if sozID == "" {
		t.Fatal("sözleşme oluşmadı")
	}
	izleyiciAccess, _ := loginSessionDevice(t, env, izleyiciEposta, testSifre, cihaz)
	cIzleyici := env.withDevice(izleyiciAccess, cihaz)
	var liste struct {
		Sozlesmeler []struct{ ID string }
	}
	cIzleyici.MustPost(`query { sozlesmeler { id } }`, &liste)
	if len(liste.Sozlesmeler) != 1 || liste.Sozlesmeler[0].ID != sozID {
		t.Fatalf("izleyici sözleşme listesi = %+v", liste.Sozlesmeler)
	}
	silErr := cIzleyici.Post(`mutation ($id: ID!) { sozlesmeSil(id: $id) }`,
		new(struct{ SozlesmeSil bool }), client.Var("id", sozID))
	if silErr == nil || !strings.Contains(silErr.Error(), "bu işlem için yetkiniz yok") {
		t.Fatalf("izleyici silme: %v", silErr)
	}
	t.Log("GORUNTULEYICI sözleşmeyi gördü, silme reddedildi")

	t.Log("4. başka organizasyonun sözleşmesi sorgulanamıyor")
	postKayitKurumsal(t, env.c, digerSahipEposta, testSifre, "Ege Turizm")
	digerDogrulama := tokenFromGovde(env.mail.lastGovde())
	if !postDogrula(t, env.c, digerDogrulama) {
		t.Fatal("diğer e-posta doğrulanamadı")
	}
	digerAccess, _ := loginSessionDevice(t, env, digerSahipEposta, testSifre, cihaz)
	cDiger := env.withDevice(digerAccess, cihaz)
	var digerSoz struct {
		Sozlesme *struct{ ID string }
	}
	cDiger.MustPost(`query ($id: ID!) { sozlesme(id: $id) { id } }`, &digerSoz, client.Var("id", sozID))
	if digerSoz.Sozlesme != nil {
		t.Fatal("başka organizasyonun sözleşmesi göründü")
	}
	cDiger.MustPost(`query { sozlesmeler { id } }`, &liste)
	for _, s := range liste.Sozlesmeler {
		if s.ID == sozID {
			t.Fatal("sözleşme listesinde yabancı kayıt var")
		}
	}
	t.Log("kiracı yalıtımı: yabancı sözleşme görünmüyor")

	t.Log("5. sahip silinmesi başka üye varken engellenir")
	var iste struct{ HesapSilmeIste bool }
	cSahip.MustPost(`mutation { hesapSilmeIste }`, &iste)
	if !iste.HesapSilmeIste {
		t.Fatal("hesapSilmeIste false")
	}
	silmeKodu := tokenAfterLabel(env.mail.lastGovde(), "Hesap silme onay kodunuz:")
	if silmeKodu == "" {
		t.Fatal("silme kodu yok")
	}
	silHata := env.c.Post(`mutation ($t: String!) { hesapSil(token: $t) }`,
		new(struct{ HesapSil bool }), client.Var("t", silmeKodu))
	if silHata == nil || !strings.Contains(silHata.Error(), "önce sahipliği devredin veya organizasyonu silin") {
		t.Fatalf("sahip silme: %v", silHata)
	}
	if _, err := env.users.GetByEposta(ctx, sahipEposta); err != nil {
		t.Fatal("sahip hesabı silinmiş")
	}
	t.Log("sahip silme başka üye varken reddedildi")
}

func postKayitKurumsal(t *testing.T, c *client.Client, eposta, sifre, orgAd string) {
	t.Helper()
	var out kayitYaniti
	c.MustPost(`mutation ($e: String!, $s: String!, $ad: String!) {
		kayitOl(eposta: $e, sifre: $s, hesapTipi: KURUMSAL, organizasyonAdi: $ad) { basarili mesaj }
	}`, &out, client.Var("e", eposta), client.Var("s", sifre), client.Var("ad", orgAd))
	if !out.KayitOl.Basarili {
		t.Fatal("kurumsal kayıt başarısız")
	}
}
