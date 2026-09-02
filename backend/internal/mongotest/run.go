package mongotest

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	appmongo "github.com/oz-fatma/kontrata/backend/internal/mongo"
)

// Run entegrasyon testlerini ayrı veritabanında çalıştırır; başta ve sonda koleksiyonları siler.
func Run(m *testing.M) int {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		return m.Run()
	}
	if _, err := appmongo.UseTestDatabase(); err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c, err := appmongo.Connect(ctx, uri)
	if err != nil {
		return m.Run()
	}
	defer func() {
		if err := c.Disconnect(context.Background()); err != nil {
			log.Printf("test veritabanı bağlantısı kapatılamadı: %v", err)
		}
	}()
	if err := appmongo.ResetTestDatabase(ctx, c); err != nil {
		panic(err)
	}
	code := m.Run()
	resetCtx, resetCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer resetCancel()
	if err := appmongo.ResetTestDatabase(resetCtx, c); err != nil {
		panic(err)
	}
	return code
}
