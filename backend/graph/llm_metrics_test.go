package graph

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/llm"
	"github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

func TestLLMMetrics_SinceRunStart(t *testing.T) {
	dbName := fmt.Sprintf("%s_llmmet_%s", mongo.TestDatabasePrefix, bson.NewObjectID().Hex())
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
	const cihaz = "llm-metrik-cihaz"
	postRegisterCorporate(t, env.c, eposta, testPassword, "Metrik Otel")
	if !postVerify(t, env.c, tokenFromBody(env.mail.lastBody())) {
		t.Fatal("e-posta doğrulanamadı")
	}
	access, _ := loginSessionDevice(t, env, eposta, testPassword, cihaz)
	c := env.withDevice(access, cihaz)

	sahip, err := env.users.GetByEmail(ctx, eposta)
	if err != nil {
		t.Fatal(err)
	}
	calls := repository.NewLLMCallRepository(env.db)
	old := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < 3; i++ {
		if err := calls.Insert(ctx, &repository.LLMCall{
			OrgID:      sahip.OrganizationID,
			Agent:      llm.AgentReader,
			Endpoint:   llm.EndpointUC1,
			Start:      old,
			End:        old.Add(time.Second),
			DurationMs: 1000,
			Success:    false,
			ErrorType:  llm.HataHTTP5xx,
			Attempt:    1,
		}); err != nil {
			t.Fatalf("eski kayıt: %v", err)
		}
	}
	runStart := time.Now().UTC()
	now := runStart.Add(time.Second)
	if err := calls.Insert(ctx, &repository.LLMCall{
		OrgID:      sahip.OrganizationID,
		Agent:      llm.AgentReader,
		Endpoint:   llm.EndpointUC1,
		Start:      now,
		End:        now.Add(time.Second),
		DurationMs: 800,
		Success:    true,
		ErrorType:  llm.HataYok,
		Attempt:    1,
	}); err != nil {
		t.Fatalf("yeni kayıt: %v", err)
	}

	var kosu struct {
		LlmMetrikleri struct {
			ToplamCagri    int32
			BasarisizCagri int32
			HataDagilimi   []struct {
				HataTipi string
				Adet     int32
			}
		}
	}
	c.MustPost(`query ($t: Time!) {
  llmMetrikleri(baslangic: $t) {
    toplamCagri basarisizCagri hataDagilimi { hataTipi adet }
  }
}`, &kosu, client.Var("t", runStart))
	if kosu.LlmMetrikleri.ToplamCagri != 1 || kosu.LlmMetrikleri.BasarisizCagri != 0 {
		t.Fatalf("koşu metrik = %+v", kosu.LlmMetrikleri)
	}
	if len(kosu.LlmMetrikleri.HataDagilimi) != 0 {
		t.Fatalf("önceki koşu hataları sızdı: %+v", kosu.LlmMetrikleri.HataDagilimi)
	}

	var gun struct {
		LlmMetrikleri struct{ ToplamCagri int32 }
	}
	c.MustPost(`query { llmMetrikleri { toplamCagri } }`, &gun)
	if gun.LlmMetrikleri.ToplamCagri != 4 {
		t.Fatalf("24 saat metrik = %d", gun.LlmMetrikleri.ToplamCagri)
	}
}
