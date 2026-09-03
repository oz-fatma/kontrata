package graph

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/agent"
	"github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

func TestPromptAdminFlow(t *testing.T) {
	dbName := fmt.Sprintf("%s_prompt_%s", mongo.TestDatabasePrefix, bson.NewObjectID().Hex())
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

	sahipEposta := uniqueEmail()
	yoneticiEposta := uniqueEmail()
	const cihaz = "prompt-admin-cihaz"
	postRegisterCorporate(t, env.c, sahipEposta, testPassword, "Prompt Otel")
	if !postVerify(t, env.c, tokenFromBody(env.mail.lastBody())) {
		t.Fatal("e-posta doğrulanamadı")
	}
	sahipAccess, _ := loginSessionDevice(t, env, sahipEposta, testPassword, cihaz)
	cSahip := env.withDevice(sahipAccess, cihaz)

	var met struct {
		LlmMetrikleri struct {
			ToplamCagri    int32
			OrtalamaSureMs float64
			P95SureMs      float64
		}
	}
	cSahip.MustPost(`query { llmMetrikleri { toplamCagri ortalamaSureMs p95SureMs } }`, &met)
	if met.LlmMetrikleri.ToplamCagri != 0 {
		t.Fatalf("boş metrik = %+v", met.LlmMetrikleri)
	}

	var aktif struct {
		AktifPrompt struct {
			ID                   string
			Tip                  string
			Icerik               string
			Surum                int32
			Aktif                bool
			OlusturmaTarihi      string
			OlusturanKullaniciID string
		}
	}
	cSahip.MustPost(`query { aktifPrompt(tip: OKUYUCU) { id tip icerik surum aktif olusturmaTarihi olusturanKullaniciId } }`, &aktif)
	if aktif.AktifPrompt.ID != "varsayilan" || aktif.AktifPrompt.Surum != 0 {
		t.Fatalf("varsayılan beklenirdi: %+v", aktif.AktifPrompt)
	}
	if aktif.AktifPrompt.Icerik != agent.SYSTEM_PROMPT {
		t.Fatal("varsayılan okuyucu prompt'u kod metni değil")
	}
	if aktif.AktifPrompt.OlusturmaTarihi == "" {
		t.Fatal("varsayılan prompt olusturmaTarihi boş")
	}
	if aktif.AktifPrompt.OlusturanKullaniciID != "varsayilan" {
		t.Fatalf("varsayılan oluşturan = %q", aktif.AktifPrompt.OlusturanKullaniciID)
	}

	var listeBos struct {
		PromptSurumleri []struct {
			ID    string
			Surum int32
		}
	}
	cSahip.MustPost(`query { promptSurumleri(tip: OKUYUCU) { id surum } }`, &listeBos)
	if listeBos.PromptSurumleri == nil {
		t.Fatal("boş sürüm listesi null dönmemeli")
	}
	if len(listeBos.PromptSurumleri) != 0 {
		t.Fatalf("kayıtlı sürüm yokken liste = %+v", listeBos.PromptSurumleri)
	}

	const v1 = "OKUYUCU SURUM BIR yalniz JSON uret."
	var guncelle struct {
		PromptGuncelle struct {
			ID     string
			Surum  int32
			Aktif  bool
			Icerik string
		}
	}
	cSahip.MustPost(`mutation ($t: PromptTipi!, $i: String!) { promptGuncelle(tip: $t, icerik: $i) { id surum aktif icerik } }`,
		&guncelle, client.Var("t", "OKUYUCU"), client.Var("i", v1))
	if guncelle.PromptGuncelle.Surum != 1 || !guncelle.PromptGuncelle.Aktif || guncelle.PromptGuncelle.Icerik != v1 {
		t.Fatalf("v1 = %+v", guncelle.PromptGuncelle)
	}
	id1 := guncelle.PromptGuncelle.ID

	const v2 = "OKUYUCU SURUM IKI yalniz JSON uret."
	cSahip.MustPost(`mutation ($t: PromptTipi!, $i: String!) { promptGuncelle(tip: $t, icerik: $i) { id surum aktif icerik } }`,
		&guncelle, client.Var("t", "OKUYUCU"), client.Var("i", v2))
	if guncelle.PromptGuncelle.Surum != 2 || !guncelle.PromptGuncelle.Aktif {
		t.Fatalf("v2 = %+v", guncelle.PromptGuncelle)
	}

	var liste struct {
		PromptSurumleri []struct {
			ID    string
			Surum int32
			Aktif bool
		}
	}
	cSahip.MustPost(`query { promptSurumleri(tip: OKUYUCU) { id surum aktif } }`, &liste)
	if len(liste.PromptSurumleri) != 2 {
		t.Fatalf("sürüm sayısı = %d", len(liste.PromptSurumleri))
	}
	aktifSayi := 0
	for _, p := range liste.PromptSurumleri {
		if p.Aktif {
			aktifSayi++
			if p.Surum != 2 {
				t.Fatalf("aktif sürüm = %d", p.Surum)
			}
		}
	}
	if aktifSayi != 1 {
		t.Fatalf("aktif sayısı = %d", aktifSayi)
	}

	kayitlar, err := env.audit.ListByUser(ctx, mustUserID(t, env, sahipEposta))
	if err != nil {
		t.Fatalf("denetim: %v", err)
	}
	if !auditHas(kayitlar, repository.EventPromptUpdated) {
		t.Fatal("PROMPT_GUNCELLENDI yok")
	}
	for _, rec := range kayitlar {
		if rec.Event == repository.EventPromptUpdated && strings.Contains(rec.Detail, v1) {
			t.Fatal("denetimde prompt içeriği var")
		}
	}

	var don struct {
		PromptSurumeDon struct {
			Surum  int32
			Aktif  bool
			Icerik string
		}
	}
	cSahip.MustPost(`mutation ($id: ID!) { promptSurumeDon(id: $id) { surum aktif icerik } }`,
		&don, client.Var("id", id1))
	if don.PromptSurumeDon.Surum != 1 || !don.PromptSurumeDon.Aktif || don.PromptSurumeDon.Icerik != v1 {
		t.Fatalf("geri dönüş = %+v", don.PromptSurumeDon)
	}
	if !auditHas(kayitlarAfter(t, env, sahipEposta), repository.EventPromptReverted) {
		t.Fatal("PROMPT_SURUME_DONULDU yok")
	}

	var ayar struct {
		Ayarlar struct {
			DenetciRiskEsigi float64
			MaxToken         int32
			GuncellemeTarihi string
		}
	}
	cSahip.MustPost(`query { ayarlar { denetciRiskEsigi maxToken guncellemeTarihi } }`, &ayar)
	if ayar.Ayarlar.DenetciRiskEsigi != 0.75 || ayar.Ayarlar.MaxToken != 600 {
		t.Fatalf("varsayılan ayar = %+v", ayar.Ayarlar)
	}
	if ayar.Ayarlar.GuncellemeTarihi == "" {
		t.Fatal("ayarlar.guncellemeTarihi boş; Time! null basmış")
	}
	sahipDoc, err := env.users.GetByEmail(ctx, sahipEposta)
	if err != nil {
		t.Fatal(err)
	}
	kayit, err := env.settings.GetByOrg(ctx, sahipDoc.OrganizationID)
	if err != nil {
		t.Fatalf("ayarlar belgesi yazılmalıydı: %v", err)
	}
	if kayit.UpdatedAt.IsZero() {
		t.Fatal("yazılan ayarın guncellemeTarihi sıfır")
	}
	var ayarYaz struct {
		AyarlariGuncelle struct {
			DenetciRiskEsigi float64
			MaxToken         int32
		}
	}
	cSahip.MustPost(`mutation { ayarlariGuncelle(denetciRiskEsigi: 0.5, maxToken: 400) { denetciRiskEsigi maxToken } }`, &ayarYaz)
	if ayarYaz.AyarlariGuncelle.DenetciRiskEsigi != 0.5 || ayarYaz.AyarlariGuncelle.MaxToken != 400 {
		t.Fatalf("ayar yaz = %+v", ayarYaz.AyarlariGuncelle)
	}
	if !auditHas(kayitlarAfter(t, env, sahipEposta), repository.EventSettingsUpdated) {
		t.Fatal("AYARLAR_GUNCELLENDI yok")
	}

	cSahip.MustPost(`mutation ($e: String!, $r: Rol!) { uyeDavetEt(eposta: $e, rol: $r) }`,
		new(struct{ UyeDavetEt bool }), client.Var("e", yoneticiEposta), client.Var("r", "YONETICI"))
	kod := tokenAfterLabel(env.mail.lastBody(), "Davet kodunuz:")
	env.c.MustPost(`mutation ($t: String!, $s: String!) { davetiKabulEt(token: $t, sifre: $s) }`,
		new(struct{ DavetiKabulEt bool }), client.Var("t", kod), client.Var("s", testPassword))
	yoneticiAccess, _ := loginSessionDevice(t, env, yoneticiEposta, testPassword, cihaz)
	cYonetici := env.withDevice(yoneticiAccess, cihaz)
	err = cYonetici.Post(`mutation ($t: PromptTipi!, $i: String!) { promptGuncelle(tip: $t, icerik: $i) { id } }`,
		new(struct{ PromptGuncelle struct{ ID string } }), client.Var("t", "OKUYUCU"), client.Var("i", "yasak"))
	if err == nil || !strings.Contains(err.Error(), "yetkiniz yok") {
		t.Fatalf("yönetici prompt güncellemeli değildi: %v", err)
	}
	err = cYonetici.Post(`query { llmMetrikleri { toplamCagri } }`,
		new(struct{ LlmMetrikleri struct{ ToplamCagri int32 } }))
	if err == nil || !strings.Contains(err.Error(), "yetkiniz yok") {
		t.Fatalf("yönetici metrik görmemeli: %v", err)
	}

	sahip, err := env.users.GetByEmail(ctx, sahipEposta)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	doc := repository.Contract{
		UserID:         sahip.ID,
		OrganizationID: sahip.OrganizationID,
		CreatedAt:      now,
		UpdatedAt:      now,
		Status:         "YUKLENDI",
		RoomAllotments: []repository.RoomAllotment{},
		Prices:         []repository.Price{},
		StopSale:       []repository.StopSaleRange{},
	}
	if err := env.soz.Create(ctx, &doc); err != nil {
		t.Fatalf("sözleşme: %v", err)
	}
	env.sozSvc.AttachExtract(env.files, &recordingLLM{out: "{}"}, "")
	env.sozSvc.RunExtractForTest(doc.ID.Hex())
	got, err := env.soz.GetByID(ctx, doc.ID.Hex())
	if err != nil {
		t.Fatal(err)
	}
	if got.PromptVersion == nil || *got.PromptVersion != 1 {
		t.Fatalf("promptSurumu = %v (v1 aktif olmalı)", got.PromptVersion)
	}
}

func mustUserID(t *testing.T, env registerEnv, eposta string) bson.ObjectID {
	t.Helper()
	u, err := env.users.GetByEmail(context.Background(), eposta)
	if err != nil {
		t.Fatal(err)
	}
	return u.ID
}

func kayitlarAfter(t *testing.T, env registerEnv, eposta string) []repository.AuditRecord {
	t.Helper()
	list, err := env.audit.ListByUser(context.Background(), mustUserID(t, env, eposta))
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func auditHas(recs []repository.AuditRecord, event string) bool {
	for _, r := range recs {
		if r.Event == event {
			return true
		}
	}
	return false
}

type recordingLLM struct {
	out    string
	system []string
	user   []string
}

func (r *recordingLLM) Generate(_ context.Context, sys, user string) (string, error) {
	r.system = append(r.system, sys)
	r.user = append(r.user, user)
	if r.out == "" {
		return "{}", nil
	}
	return r.out, nil
}
