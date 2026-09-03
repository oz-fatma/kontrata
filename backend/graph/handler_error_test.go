package graph

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/service"
)

func TestPresentGraphQLError(t *testing.T) {
	ctx := context.Background()

	got := presentGraphQLError(ctx, auth.ErrForbidden, false)
	if got.Message != auth.ErrForbidden.Error() {
		t.Fatalf("yetki mesajı = %q", got.Message)
	}

	got = presentGraphQLError(ctx, service.ErrEmptyPrompt, false)
	if got.Message != service.ErrEmptyPrompt.Error() {
		t.Fatalf("prompt mesajı = %q", got.Message)
	}

	internal := errors.New("cannot return null for non-nullable field Ayarlar.guncellemeTarihi")
	got = presentGraphQLError(ctx, internal, false)
	if got.Message != "işlem tamamlanamadı" {
		t.Fatalf("üretim mesajı = %q", got.Message)
	}
	if got.Extensions != nil {
		t.Fatal("uzantı sızmamalı")
	}

	got = presentGraphQLError(ctx, internal, true)
	if !strings.Contains(got.Message, "cannot return null") {
		t.Fatalf("playground mesajı gizlendi: %q", got.Message)
	}
}
