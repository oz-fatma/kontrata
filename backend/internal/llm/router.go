package llm

import (
	"context"
	"log"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	latencyWindow       = 10
	unhealthyAfter      = 3
	defaultUnhealthyFor = 60 * time.Second
	EndpointUC1         = "uc1"
	EndpointUC2         = "uc2"
)

// Recorder izleme kaydını yazar. Hata işi durdurmaz.
type Recorder interface {
	Record(ctx context.Context, rec CallRecord)
}

// CallRecord Mongo'ya gidecek izleme satırıdır. Metin alanı yoktur.
type CallRecord struct {
	OrgID         string
	ContractID    string
	Agent         string
	Endpoint      string
	Start         time.Time
	End           time.Time
	DurationMs    int64
	InChars       int
	OutChars      int
	Success       bool
	ErrorType     string
	Attempt       int
	PromptVersion *int32
}

// NopRecorder kayıt yazmaz.
type NopRecorder struct{}

func (NopRecorder) Record(context.Context, CallRecord) {}

// NamedClient yönlendiricideki bir uçtur.
type NamedClient struct {
	Name   string
	Client Client
}

type endpointState struct {
	name            string
	client          Client
	active          int
	latencies       []time.Duration
	unhealthy       bool
	unhealthyAt     time.Time
	unhealthyUntil  time.Time
	consecutiveFail int
	probing         bool
	total           int64
	failed          int64
}

// Router kayıtlı uçlar arasında istek dağıtır.
type Router struct {
	mu       sync.Mutex
	eps      []*endpointState
	rec      Recorder
	now      func() time.Time
	cooldown time.Duration
}

// NewRouter verilen uçlarla yönlendirici kurar. Boş dilimde Generate unavailable döner.
func NewRouter(eps []NamedClient, rec Recorder) *Router {
	if rec == nil {
		rec = NopRecorder{}
	}
	r := &Router{
		rec:      rec,
		now:      time.Now,
		cooldown: defaultUnhealthyFor,
	}
	for _, e := range eps {
		if e.Client == nil || e.Name == "" {
			continue
		}
		r.eps = append(r.eps, &endpointState{name: e.Name, client: e.Client})
	}
	return r
}

// Generate bir uç seçer, çağırır, izler. Seçim sırası: sağlıklı, az aktif, düşük gecikme.
func (r *Router) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return r.generate(ctx, systemPrompt, userPrompt, 0)
}

func (r *Router) generate(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	if r == nil || len(r.eps) == 0 {
		return "", ErrUnavailable
	}
	ep := r.acquire()
	start := r.now()
	var seen int
	ctx = withAttemptSink(ctx, func(a Attempt) {
		seen++
		r.noteAttempt(ep, a)
		r.emit(ctx, ep.name, a)
	})
	client := ep.client
	if maxTokens > 0 {
		client = LimitTokens(client, maxTokens)
	}
	out, err := client.Generate(ctx, systemPrompt, userPrompt)
	if seen == 0 {
		end := r.now()
		tipi, cold := classifyError(err)
		a := Attempt{
			Number:    1,
			Start:     start,
			End:       end,
			InChars:   utf8.RuneCountInString(BuildChatPrompt(systemPrompt, userPrompt)),
			OutChars:  utf8.RuneCountInString(out),
			Success:   err == nil,
			ErrorType: tipi,
			Cold:      cold,
		}
		if a.Success {
			a.ErrorType = HataYok
		}
		r.noteAttempt(ep, a)
		r.emit(ctx, ep.name, a)
	}
	dur := r.now().Sub(start)
	aktif := r.release(ep, dur, err)
	log.Printf("llm cagri uc=%s gecikme=%s aktif=%d", ep.name, dur, aktif)
	return out, err
}

func (r *Router) acquire() *endpointState {
	r.mu.Lock()
	defer r.mu.Unlock()
	ep := r.selectLocked()
	ep.active++
	now := r.now()
	if ep.unhealthy && !now.Before(ep.unhealthyUntil) {
		ep.probing = true
	}
	return ep
}

func (r *Router) selectLocked() *endpointState {
	now := r.now()
	var ready []*endpointState
	for _, ep := range r.eps {
		if !ep.unhealthy {
			ready = append(ready, ep)
			continue
		}
		if !now.Before(ep.unhealthyUntil) && !ep.probing {
			ready = append(ready, ep)
		}
	}
	if len(ready) == 0 {
		return oldestUnhealthy(r.eps)
	}
	best := ready[0]
	for _, ep := range ready[1:] {
		if betterEndpoint(ep, best) {
			best = ep
		}
	}
	return best
}

func oldestUnhealthy(eps []*endpointState) *endpointState {
	best := eps[0]
	for _, ep := range eps[1:] {
		if ep.unhealthyAt.IsZero() && !best.unhealthyAt.IsZero() {
			continue
		}
		if best.unhealthyAt.IsZero() || ep.unhealthyAt.Before(best.unhealthyAt) {
			best = ep
		}
	}
	return best
}

func betterEndpoint(a, b *endpointState) bool {
	if a.active != b.active {
		return a.active < b.active
	}
	avgA, hasA := avgLatency(a.latencies)
	avgB, hasB := avgLatency(b.latencies)
	if hasA != hasB {
		return !hasA
	}
	if hasA && avgA != avgB {
		return avgA < avgB
	}
	return a.name < b.name
}

func avgLatency(in []time.Duration) (time.Duration, bool) {
	if len(in) == 0 {
		return 0, false
	}
	var sum time.Duration
	for _, d := range in {
		sum += d
	}
	return sum / time.Duration(len(in)), true
}

func (r *Router) noteAttempt(ep *endpointState, a Attempt) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ep.total++
	if a.Success {
		ep.consecutiveFail = 0
		ep.unhealthy = false
		ep.probing = false
		return
	}
	if a.Cold {
		return
	}
	ep.failed++
	if ep.probing {
		ep.unhealthy = true
		ep.unhealthyAt = r.now()
		ep.unhealthyUntil = r.now().Add(r.cooldown)
		ep.probing = false
		ep.consecutiveFail = unhealthyAfter
		return
	}
	ep.consecutiveFail++
	if ep.consecutiveFail >= unhealthyAfter {
		ep.unhealthy = true
		ep.unhealthyAt = r.now()
		ep.unhealthyUntil = r.now().Add(r.cooldown)
	}
}

func (r *Router) release(ep *endpointState, dur time.Duration, err error) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ep.active > 0 {
		ep.active--
	}
	aktif := ep.active
	if err == nil {
		ep.latencies = append(ep.latencies, dur)
		if len(ep.latencies) > latencyWindow {
			ep.latencies = ep.latencies[len(ep.latencies)-latencyWindow:]
		}
	}
	return aktif + 1
}

func (r *Router) emit(ctx context.Context, uc string, a Attempt) {
	if r.rec == nil {
		return
	}
	meta, _ := MetaFrom(ctx)
	tipi := a.ErrorType
	if tipi == "" {
		tipi = HataYok
	}
	rec := CallRecord{
		OrgID:         meta.OrgID,
		ContractID:    meta.ContractID,
		Agent:         meta.Agent,
		Endpoint:      uc,
		Start:         a.Start,
		End:           a.End,
		DurationMs:    a.End.Sub(a.Start).Milliseconds(),
		InChars:       a.InChars,
		OutChars:      a.OutChars,
		Success:       a.Success,
		ErrorType:     tipi,
		Attempt:       a.Number,
		PromptVersion: meta.PromptVersion,
	}
	r.rec.Record(ctx, rec)
}

type limitedRouter struct {
	r *Router
	n int
}

func (l *limitedRouter) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if l == nil || l.r == nil {
		return "", ErrUnavailable
	}
	return l.r.generate(ctx, systemPrompt, userPrompt, l.n)
}
