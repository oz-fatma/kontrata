package llm

import (
	"context"
	"time"
)

type metaKey struct{}
type attemptKey struct{}
type temperatureKey struct{}

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

// WithTemperature üretim sıcaklığını bağlama yazar. Sıfır veya negatif yok sayılır.
func WithTemperature(ctx context.Context, t float64) context.Context {
	if t <= 0 {
		return ctx
	}
	return context.WithValue(ctx, temperatureKey{}, t)
}

// TemperatureFrom bağlamdaki sıcaklığı okur.
func TemperatureFrom(ctx context.Context) (float64, bool) {
	t, ok := ctx.Value(temperatureKey{}).(float64)
	if !ok || t <= 0 {
		return 0, false
	}
	return t, true
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
