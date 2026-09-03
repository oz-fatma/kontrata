package llm

import (
	"context"
	"time"
)

type metaKey struct{}
type attemptKey struct{}

const (
	AgentReader  = "OKUYUCU"
	AgentAuditor = "DENETCI"

	HataYok     = "yok"
	HataTimeout = "zaman_asimi"
	HataHTTP5xx = "http_5xx"
	HataHTTP4xx = "http_4xx"
	HataParse   = "ayristirma"
)

// Meta izleme kaydı için çağrı bağlamıdır. Prompt ve çıktı taşımaz.
type Meta struct {
	OrgID         string
	ContractID    string
	Agent         string
	PromptVersion *int32
}

// WithMeta izleme alanlarını bağlama yazar.
func WithMeta(ctx context.Context, m Meta) context.Context {
	return context.WithValue(ctx, metaKey{}, m)
}

// MetaFrom bağlamdaki izleme alanlarını okur.
func MetaFrom(ctx context.Context) (Meta, bool) {
	m, ok := ctx.Value(metaKey{}).(Meta)
	return m, ok
}

// Attempt tek bir HTTP denemesinin özetidir. Gövde içermez.
type Attempt struct {
	Number    int
	Start     time.Time
	End       time.Time
	InChars   int
	OutChars  int
	Success   bool
	ErrorType string
	Cold      bool
}

func withAttemptSink(ctx context.Context, fn func(Attempt)) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, attemptKey{}, fn)
}

func emitAttempt(ctx context.Context, a Attempt) {
	fn, ok := ctx.Value(attemptKey{}).(func(Attempt))
	if !ok || fn == nil {
		return
	}
	fn(a)
}
