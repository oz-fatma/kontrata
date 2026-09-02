package graph

import (
	"context"

	"github.com/99designs/gqlgen/graphql"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
)

// AuthDirective @auth alanlarında kimlik doğrulaması ister.
func AuthDirective(ctx context.Context, _ any, next graphql.Resolver) (any, error) {
	if _, ok := auth.IdentityFrom(ctx); !ok {
		return nil, auth.ErrUnauthorized
	}
	return next(ctx)
}
