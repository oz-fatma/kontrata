package graph

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"

	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

func TestYeniCihazKaydiVeEposta(t *testing.T) {
	ctx, env := setupKayit(t)
	eposta := uniqueEposta()
	registerVerified(t, env, eposta, testSifre)
	user, err := env.users.GetByEposta(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	access, _ := loginSessionDevice(t, env, eposta, testSifre, "cihaz-a")
	if env.mail.countKonu("Yeni cihazdan giriş") != 1 {
		t.Fatal("yeni cihaz e-postası yok")
	}
	if !strings.Contains(env.mail.lastGovde(), "hesabınıza yeni bir cihazdan giriş yapıldı") {
		t.Fatal("bilgi e-postası metni yok")
	}
	n, err := env.devices.CountByUser(ctx, user.ID)
	if err != nil || n != 1 {
		t.Fatalf("cihaz sayısı = %d err=%v", n, err)
	}
	docs, err := env.devices.ListByUser(ctx, user.ID)
	if err != nil || len(docs) != 1 || docs[0].Guvenilir {
		t.Fatal("yeni cihaz güvenilir kaydedildi")
	}
	c := env.withToken(access)
	var list struct {
		Cihazlarim []struct{ ID, Ad string }
	}
	c.MustPost(`query { cihazlarim { id ad } }`, &list)
	if len(list.Cihazlarim) != 1 {
		t.Fatal("cihazlarim boş")
	}
}

func TestAyniCihazIkinciGirisGunceller(t *testing.T) {
	ctx, env := setupKayit(t)
	eposta := uniqueEposta()
	registerVerified(t, env, eposta, testSifre)
	user, err := env.users.GetByEposta(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	loginSessionDevice(t, env, eposta, testSifre, "cihaz-a")
	first, err := env.devices.ListByUser(ctx, user.ID)
	if err != nil || len(first) != 1 {
		t.Fatal("ilk cihaz yok")
	}
	ilk := first[0].IlkGorulme
	son := first[0].SonGorulme
	time.Sleep(20 * time.Millisecond)
	loginSessionDevice(t, env, eposta, testSifre, "cihaz-a")
	if env.mail.countKonu("Yeni cihazdan giriş") != 1 {
		t.Fatal("ikinci girişte yeni cihaz e-postası gitti")
	}
	again, err := env.devices.ListByUser(ctx, user.ID)
	if err != nil || len(again) != 1 {
		t.Fatalf("cihaz sayısı = %d", len(again))
	}
	if !again[0].IlkGorulme.Equal(ilk) {
		t.Fatal("ilkGorulme değişti")
	}
	if !again[0].SonGorulme.After(son) {
		t.Fatal("sonGorulme güncellenmedi")
	}
}

func TestCihazKaldirOturumlariIptalEder(t *testing.T) {
	_, env := setupKayit(t)
	eposta := uniqueEposta()
	registerVerified(t, env, eposta, testSifre)
	accessA, refreshA := loginSessionDevice(t, env, eposta, testSifre, "cihaz-a")
	cA := env.withDevice(accessA, "cihaz-a")
	var listA struct {
		Cihazlarim []struct{ ID string }
	}
	cA.MustPost(`query { cihazlarim { id } }`, &listA)
	if len(listA.Cihazlarim) != 1 {
		t.Fatal("ilk cihaz yok")
	}
	idA := listA.Cihazlarim[0].ID
	accessB, _ := loginSessionDevice(t, env, eposta, testSifre, "cihaz-b")
	cB := env.withDevice(accessB, "cihaz-b")
	var sil struct{ CihazKaldir bool }
	cB.MustPost(`mutation ($id: ID!) { cihazKaldir(id: $id) }`, &sil, client.Var("id", idA))
	if !sil.CihazKaldir {
		t.Fatal("cihazKaldir false")
	}
	var yenile struct {
		JetonYenile struct{ ErisimJetonu string }
	}
	err := env.c.Post(`mutation ($r: String!) {
		jetonYenile(yenilemeJetonu: $r) { erisimJetonu }
	}`, &yenile, client.Var("r", refreshA))
	if err == nil {
		t.Fatal("kaldırılan cihazın oturumu hâlâ yenileniyor")
	}
	var again struct {
		Cihazlarim []struct{ ID string }
	}
	cB.MustPost(`query { cihazlarim { id } }`, &again)
	if len(again.Cihazlarim) != 1 || again.Cihazlarim[0].ID == idA {
		t.Fatal("yanlış cihaz kaldı")
	}
}

func TestHesapSilmeZinciri(t *testing.T) {
	ctx, env := setupKayit(t)
	if !env.db.ReplicaSet(ctx) {
		t.Skip("hesap silme atomik işlem için replica set gerekli")
	}
	eposta := uniqueEposta()
	registerVerified(t, env, eposta, testSifre)
	user, err := env.users.GetByEposta(ctx, eposta)
	if err != nil {
		t.Fatalf("kullanıcı okunamadı")
	}
	access, _ := loginSession(t, env, eposta, testSifre)
	c := env.withToken(access)
	c.MustPost(`mutation ($g: SozlesmeGirdi!) { sozlesmeOlustur(girdi: $g) { id } }`,
		new(struct{ SozlesmeOlustur struct{ ID string } }), client.Var("g", sozlesmeGirdi()))

	var iste struct{ HesapSilmeIste bool }
	c.MustPost(`mutation { hesapSilmeIste }`, &iste)
	if !iste.HesapSilmeIste {
		t.Fatal("hesapSilmeIste false")
	}
	plain := tokenAfterLabel(env.mail.lastGovde(), "Hesap silme onay kodunuz:")
	if plain == "" {
		t.Fatal("silme kodu yok")
	}
	nSoz, _ := env.soz.CountByUser(ctx, user.ID)
	nCihaz, _ := env.devices.CountByUser(ctx, user.ID)
	nOturum, _ := env.sessions.CountByUser(ctx, user.ID)
	if nSoz < 1 || nCihaz < 1 || nOturum < 1 {
		t.Fatal("silinecek kayıt yok")
	}
	var sil struct{ HesapSil bool }
	env.c.MustPost(`mutation ($t: String!) { hesapSil(token: $t) }`, &sil, client.Var("t", plain))
	if !sil.HesapSil {
		t.Fatal("hesapSil false")
	}
	if _, err := env.users.GetByEposta(ctx, eposta); err == nil {
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
	kayit, err := env.audit.Latest(ctx, repository.OlayHesapSilindi)
	if err != nil {
		t.Fatal("HESAP_SILINDI yok")
	}
	s, ok := kayit.KullaniciID.(string)
	if !ok || s != repository.KullaniciSilinmis {
		t.Fatalf("kullaniciId = %#v", kayit.KullaniciID)
	}
	if kayit.IPAdresi != "" || kayit.KullaniciAjani != "" {
		t.Fatal("silme kaydında IP/UA duruyor")
	}
}

func TestVerilerimiIndirHashYok(t *testing.T) {
	_, env := setupKayit(t)
	eposta := uniqueEposta()
	registerVerified(t, env, eposta, testSifre)
	access, _ := loginSession(t, env, eposta, testSifre)
	c := env.withToken(access)
	var out struct{ VerilerimiIndir string }
	c.MustPost(`query { verilerimiIndir }`, &out)
	if out.VerilerimiIndir == "" {
		t.Fatal("boş dışa aktarım")
	}
	if strings.Contains(out.VerilerimiIndir, "sifreHash") || strings.Contains(out.VerilerimiIndir, "yenilemeTokenHash") {
		t.Fatal("hash dışa aktarıldı")
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out.VerilerimiIndir), &parsed); err != nil {
		t.Fatal("JSON değil")
	}
	if parsed["eposta"] != eposta {
		t.Fatalf("eposta = %v", parsed["eposta"])
	}
}
