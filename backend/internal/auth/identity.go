package auth

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type identityKey struct{}

// Identity doğrulanmış isteğin kullanici ve oturum kimliğidir.
type Identity struct {
	UserID    bson.ObjectID
	SessionID bson.ObjectID
}

// WithIdentity kimliği bağlama yazar.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

// IdentityFrom bağlamdaki kimliği okur.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok
}
