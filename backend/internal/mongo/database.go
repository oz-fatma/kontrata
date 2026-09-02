package mongo

import (
	"context"
	"fmt"
	"os"
	"strings"
)

const (
	// DefaultDatabase üretim koleksiyonlarının veritabanıdır.
	DefaultDatabase = "kontrata"
	// TestDatabasePrefix testlerin yazabileceği adların ön ekidir.
	TestDatabasePrefix = "kontrata_test"
)

// DatabaseName MONGO_DATABASE ortam değişkenini okur; yoksa DefaultDatabase döner.
func DatabaseName() string {
	if n := strings.TrimSpace(os.Getenv("MONGO_DATABASE")); n != "" {
		return n
	}
	return DefaultDatabase
}

// DropDatabase verilen veritabanını siler. Üretim adı reddedilir.
func (c *Client) DropDatabase(ctx context.Context, name string) error {
	if c == nil || c.inner == nil {
		return nil
	}
	if !IsTestDatabase(name) {
		return fmt.Errorf("üretim veritabanı silinemez")
	}
	return c.inner.Database(name).Drop(ctx)
}

// IsTestDatabase adın test önekiyle başladığını doğrular.
func IsTestDatabase(name string) bool {
	return strings.HasPrefix(name, TestDatabasePrefix)
}
