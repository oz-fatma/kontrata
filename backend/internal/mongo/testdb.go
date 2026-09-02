package mongo

import (
	"context"
	"fmt"
	"os"
)

// UseTestDatabase MONGO_DATABASE boşsa veya üretim adıysa kontrata_test_<pid> atar.
func UseTestDatabase() (string, error) {
	name := DatabaseName()
	if name == "" || name == DefaultDatabase {
		name = fmt.Sprintf("%s_%d", TestDatabasePrefix, os.Getpid())
		if err := os.Setenv("MONGO_DATABASE", name); err != nil {
			return "", err
		}
	}
	if !IsTestDatabase(name) {
		return "", fmt.Errorf("testler üretim veritabanına yazamaz")
	}
	return name, nil
}

// ResetTestDatabase test veritabanındaki tüm koleksiyonları siler.
func ResetTestDatabase(ctx context.Context, c *Client) error {
	return c.DropDatabase(ctx, DatabaseName())
}
