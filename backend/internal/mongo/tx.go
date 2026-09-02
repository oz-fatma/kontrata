package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// WithTransaction oturum açıp fn'i atomik çalıştırır. fn hata dönerse tümü geri alınır.
// Çok belgelik silme replica set (veya mongos) gerektirir.
func (c *Client) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	if c == nil || c.inner == nil {
		return errUnreachable
	}
	sess, err := c.inner.StartSession()
	if err != nil {
		return errUnreachable
	}
	defer sess.EndSession(ctx)
	_, err = sess.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		return nil, fn(ctx)
	})
	if err != nil {
		return err
	}
	return nil
}

// ReplicaSet sunucunun işlem (transaction) kullanabileceğini döner.
func (c *Client) ReplicaSet(ctx context.Context) bool {
	if c == nil || c.inner == nil {
		return false
	}
	var out bson.M
	err := c.inner.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&out)
	if err != nil {
		return false
	}
	_, ok := out["setName"]
	return ok
}
