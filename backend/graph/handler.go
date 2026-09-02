package graph

import (
	"context"
	"errors"
	"log"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/oz-fatma/kontrata/backend/internal/repository"
	"github.com/oz-fatma/kontrata/backend/internal/service"
)

// RegisterRoutes /graphql ucunu ve isteğe bağlı playground'u bağlar.
func RegisterRoutes(r chi.Router, svc *service.SozlesmeService, enablePlayground bool) {
	srv := handler.New(NewExecutableSchema(Config{Resolvers: &Resolver{Service: svc}}))
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.SetRecoverFunc(func(_ context.Context, rec any) error {
		log.Printf("graphql panic kurtarıldı")
		return errors.New("iç sunucu hatası")
	})
	srv.SetErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
		err := graphql.DefaultErrorPresenter(ctx, e)
		switch {
		case errors.Is(e, repository.ErrNotFound), errors.Is(e, repository.ErrInvalidID), errors.Is(e, repository.ErrUnavailable):
			err.Message = e.Error()
		default:
			err.Message = "işlem tamamlanamadı"
		}
		err.Extensions = nil
		return err
	})
	if enablePlayground {
		srv.Use(extension.Introspection{})
		r.Handle("/playground", playground.Handler("Kontrata", "/graphql"))
	}
	r.Handle("/graphql", srv)
}
