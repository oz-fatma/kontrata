package mongo

import (
	"context"
	"errors"

	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

var errUnreachable = errors.New("veritabanına bağlanılamadı")

// Client sürücü istemcisinin ince sarmalayıcısıdır.
type Client struct {
	inner *drivermongo.Client
}

// Connect verilen URI ile bağlanır ve ctx süresi içinde Ping ile doğrular.
// Bağlantı dizesi hata değerine yazılmaz.
func Connect(ctx context.Context, uri string) (*Client, error) {
	inner, err := drivermongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, errUnreachable
	}
	if err := inner.Ping(ctx, readpref.Primary()); err != nil {
		_ = inner.Disconnect(ctx)
		return nil, errUnreachable
	}
	return &Client{inner: inner}, nil
}

// Disconnect kapanışta bağlantıyı bırakır.
func (c *Client) Disconnect(ctx context.Context) error {
	if c == nil || c.inner == nil {
		return nil
	}
	if err := c.inner.Disconnect(ctx); err != nil {
		return errUnreachable
	}
	return nil
}

// Ping sağlık denetimi için kısa süreli erişim kontrolüdür.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.inner == nil {
		return errUnreachable
	}
	if err := c.inner.Ping(ctx, readpref.Primary()); err != nil {
		return errUnreachable
	}
	return nil
}
