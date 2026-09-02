package graph

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"

	"github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
	"github.com/oz-fatma/kontrata/backend/internal/service"
)

func TestSozlesmeOlusturVeOku(t *testing.T) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		t.Skip("MONGO_URI yok")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db, err := mongo.Connect(ctx, uri)
	if err != nil {
		t.Fatalf("mongo bağlanamadı")
	}
	defer db.Disconnect(context.Background())

	repo := repository.NewSozlesmeRepository(db)
	if err := repo.EnsureIndexes(ctx); err != nil {
		t.Fatalf("indeks oluşturulamadı")
	}
	svc := service.NewSozlesmeService(repo)

	srv := handler.New(NewExecutableSchema(Config{Resolvers: &Resolver{Service: svc}}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	girdi := map[string]any{
		"dosyaAdi": "ornek-kontrat.pdf",
		"donem": map[string]any{
			"baslangic": "2026-04-01",
			"bitis":     "2026-10-31",
		},
		"odaKontenjanlari": []map[string]any{
			{"odaTipi": "standart", "adet": 12},
		},
		"fiyatlar": []map[string]any{
			{"odaTipi": "standart", "tutar": 85.5, "birim": "ODA_GECELIK", "pansiyon": "HB"},
		},
		"release": map[string]any{
			"gun":    21,
			"kapsam": "KONTENJAN_IADESI",
		},
		"stopSale": []any{},
	}

	var created struct {
		SozlesmeOlustur struct {
			ID               string
			DosyaAdi         *string
			Durum            string
			Donem            *struct{ Baslangic, Bitis *string }
			OdaKontenjanlari []struct {
				OdaTipi string
				Adet    int32
			}
			Fiyatlar []struct {
				OdaTipi string
				Tutar   float64
				Birim   string
			}
			Release *struct{ Gun int32 }
			StopSale []any
		}
	}
	c.MustPost(`mutation ($g: SozlesmeGirdi!) {
		sozlesmeOlustur(girdi: $g) {
			id dosyaAdi durum
			donem { baslangic bitis }
			odaKontenjanlari { odaTipi adet }
			fiyatlar { odaTipi tutar birim }
			release { gun }
			stopSale { baslangic }
		}
	}`, &created, client.Var("g", girdi))

	id := created.SozlesmeOlustur.ID
	if id == "" {
		t.Fatal("id boş")
	}
	t.Cleanup(func() {
		_, _ = svc.Delete(context.Background(), id)
	})

	if created.SozlesmeOlustur.DosyaAdi == nil || *created.SozlesmeOlustur.DosyaAdi != "ornek-kontrat.pdf" {
		t.Fatalf("dosyaAdi eşleşmedi")
	}
	if created.SozlesmeOlustur.Durum != "YUKLENDI" {
		t.Fatalf("durum = %s", created.SozlesmeOlustur.Durum)
	}
	if created.SozlesmeOlustur.Donem == nil || created.SozlesmeOlustur.Donem.Baslangic == nil || *created.SozlesmeOlustur.Donem.Baslangic != "2026-04-01" {
		t.Fatalf("donem.baslangic eşleşmedi")
	}
	if len(created.SozlesmeOlustur.OdaKontenjanlari) != 1 || created.SozlesmeOlustur.OdaKontenjanlari[0].Adet != 12 {
		t.Fatalf("odaKontenjanlari eşleşmedi")
	}
	if len(created.SozlesmeOlustur.Fiyatlar) != 1 || created.SozlesmeOlustur.Fiyatlar[0].Tutar != 85.5 {
		t.Fatalf("fiyatlar eşleşmedi")
	}
	if created.SozlesmeOlustur.Release == nil || created.SozlesmeOlustur.Release.Gun != 21 {
		t.Fatalf("release.gun eşleşmedi")
	}
	if created.SozlesmeOlustur.StopSale == nil {
		t.Fatal("stopSale null; boş dizi bekleniyordu")
	}
	if len(created.SozlesmeOlustur.StopSale) != 0 {
		t.Fatalf("stopSale uzunluğu = %d, beklenen 0", len(created.SozlesmeOlustur.StopSale))
	}

	var got struct {
		Sozlesme *struct {
			ID               string
			DosyaAdi         *string
			Durum            string
			Donem            *struct{ Baslangic, Bitis *string }
			OdaKontenjanlari []struct {
				OdaTipi string
				Adet    int32
			}
			Fiyatlar []struct {
				OdaTipi string
				Tutar   float64
				Birim   string
			}
			Release *struct{ Gun int32 }
			StopSale []any
		}
	}
	c.MustPost(`query ($id: ID!) {
		sozlesme(id: $id) {
			id dosyaAdi durum
			donem { baslangic bitis }
			odaKontenjanlari { odaTipi adet }
			fiyatlar { odaTipi tutar birim }
			release { gun }
			stopSale { baslangic }
		}
	}`, &got, client.Var("id", id))

	if got.Sozlesme == nil {
		t.Fatal("sozlesme null döndü")
	}
	if got.Sozlesme.ID != id {
		t.Fatalf("id eşleşmedi")
	}
	if got.Sozlesme.DosyaAdi == nil || *got.Sozlesme.DosyaAdi != "ornek-kontrat.pdf" {
		t.Fatalf("okunan dosyaAdi eşleşmedi")
	}
	if got.Sozlesme.Donem == nil || got.Sozlesme.Donem.Bitis == nil || *got.Sozlesme.Donem.Bitis != "2026-10-31" {
		t.Fatalf("okunan donem.bitis eşleşmedi")
	}
	if len(got.Sozlesme.OdaKontenjanlari) != 1 || got.Sozlesme.OdaKontenjanlari[0].OdaTipi != "standart" {
		t.Fatalf("okunan odaTipi eşleşmedi")
	}
	if len(got.Sozlesme.Fiyatlar) != 1 || got.Sozlesme.Fiyatlar[0].Birim != "ODA_GECELIK" {
		t.Fatalf("okunan birim eşleşmedi")
	}
	if got.Sozlesme.Release == nil || got.Sozlesme.Release.Gun != 21 {
		t.Fatalf("okunan release eşleşmedi")
	}
	if got.Sozlesme.StopSale == nil {
		t.Fatal("okunan stopSale null; boş dizi bekleniyordu")
	}
	if len(got.Sozlesme.StopSale) != 0 {
		t.Fatalf("okunan stopSale uzunluğu = %d, beklenen 0", len(got.Sozlesme.StopSale))
	}

	stored, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("mongo okuma başarısız")
	}
	if stored.StopSale == nil {
		t.Fatal("stopSale Mongo belgesinde yok")
	}
	if len(stored.StopSale) != 0 {
		t.Fatalf("Mongo stopSale uzunluğu = %d", len(stored.StopSale))
	}
	if len(stored.Fiyatlar) != 1 || stored.Fiyatlar[0].Birim != "oda_gecelik" {
		t.Fatalf("Mongo birim kontrat.json değeri değil")
	}
}
