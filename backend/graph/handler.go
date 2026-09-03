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

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/filestore"
	"github.com/oz-fatma/kontrata/backend/internal/llm"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
	"github.com/oz-fatma/kontrata/backend/internal/service"
)

// RegisterRoutes /graphql ucunu ve isteğe bağlı playground'u bağlar.
func RegisterRoutes(r chi.Router, svc *service.ContractService, authSvc *service.AuthService, enablePlayground bool) {
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers:  &Resolver{Service: svc, Auth: authSvc},
		Directives: DirectiveRoot{Auth: AuthDirective},
	}))
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxUploadSize: 32 << 20,
		MaxMemory:     32 << 20,
	})
	srv.SetRecoverFunc(func(_ context.Context, rec any) error {
		log.Printf("graphql panic kurtarıldı: %v", rec)
		return errors.New("iç sunucu hatası")
	})
	srv.SetErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
		err := graphql.DefaultErrorPresenter(ctx, e)
		switch {
		case errors.Is(e, repository.ErrNotFound), errors.Is(e, repository.ErrInvalidID), errors.Is(e, repository.ErrUnavailable),
			errors.Is(e, auth.ErrPasswordTooShort), errors.Is(e, auth.ErrInvalidEmail),
			errors.Is(e, auth.ErrUnauthorized), errors.Is(e, auth.ErrInvalidRefreshToken), errors.Is(e, auth.ErrMFAFailed),
			errors.Is(e, auth.ErrInvalidName), errors.Is(e, auth.ErrForbidden), errors.Is(e, auth.ErrOrgNameRequired),
			errors.Is(e, auth.ErrTransferOwnership),
			errors.Is(e, filestore.ErrNotPDF), errors.Is(e, filestore.ErrTooLarge), errors.Is(e, filestore.ErrInvalidID),
			errors.Is(e, llm.ErrUnavailable), errors.Is(e, llm.ErrColdStart),
			errors.Is(e, service.ErrApprovedReadOnly), errors.Is(e, service.ErrNotAwaitingReview),
			errors.Is(e, service.ErrUnknownField), errors.Is(e, service.ErrInvalidFieldValue):
			err.Message = e.Error()
		default:
			err.Message = "işlem tamamlanamadı"
		}
		err.Extensions = nil
		return err
	})
	if enablePlayground {
		srv.Use(extension.Introspection{})
	}
	r.Group(func(r chi.Router) {
		r.Use(auth.RequestMiddleware)
		if authSvc != nil {
			r.Use(authSvc.BearerMiddleware)
		}
		if enablePlayground {
			r.Handle("/playground", playground.Handler("Kontrata", "/graphql"))
		}
		if svc != nil {
			r.Get("/dosya/{id}", svc.ServeFile)
		}
		r.Handle("/graphql", srv)
	})
}
