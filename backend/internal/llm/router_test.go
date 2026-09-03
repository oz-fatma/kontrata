package llm

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type stubClient struct {
	name    string
	err     error
	delay   time.Duration
	started chan struct{}
	block   chan struct{}
	calls   atomic.Int32
}

func (s *stubClient) Generate(ctx context.Context, _, _ string) (string, error) {
	s.calls.Add(1)
	if s.started != nil {
		select {
		case s.started <- struct{}{}:
		default:
		}
	}
	if s.block != nil {
		select {
		case <-s.block:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	if s.err != nil {
		return "", s.err
	}
	return "ok", nil
}

func timeoutErr() error {
	return &CallError{Tipi: HataTimeout, err: ErrUnavailable}
}

func coldErr() error {
	return &CallError{Tipi: HataHTTP5xx, Cold: true, err: ErrColdStart}
}

func TestRouter_LeastActiveThenLatency(t *testing.T) {
	a := &stubClient{name: "a", started: make(chan struct{}, 8), block: make(chan struct{})}
	b := &stubClient{name: "b", started: make(chan struct{}, 8), block: make(chan struct{})}
	r := NewRouter([]NamedClient{
		{Name: EndpointUC1, Client: a},
		{Name: EndpointUC2, Client: b},
	}, nil)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = r.Generate(context.Background(), "s", "u") }()
	<-a.started
	go func() { defer wg.Done(); _, _ = r.Generate(context.Background(), "s", "u") }()
	<-b.started
	close(a.block)
	close(b.block)
	wg.Wait()
	if a.calls.Load() != 1 || b.calls.Load() != 1 {
		t.Fatalf("dağılım uc1=%d uc2=%d", a.calls.Load(), b.calls.Load())
	}
}

func TestRouter_UnhealthySkipped(t *testing.T) {
	bad := &stubClient{err: timeoutErr()}
	good := &stubClient{}
	r := NewRouter([]NamedClient{
		{Name: EndpointUC1, Client: bad},
		{Name: EndpointUC2, Client: good},
	}, nil)
	for i := 0; i < unhealthyAfter; i++ {
		_, err := r.Generate(context.Background(), "s", "u")
		if err == nil {
			t.Fatal("hata bekleniyordu")
		}
	}
	for i := 0; i < 5; i++ {
		out, err := r.Generate(context.Background(), "s", "u")
		if err != nil || out != "ok" {
			t.Fatalf("sağlıklı uca gitmeli: %v %s", err, out)
		}
	}
	if good.calls.Load() != 5 {
		t.Fatalf("iyi uç çağrı = %d", good.calls.Load())
	}
	if bad.calls.Load() != int32(unhealthyAfter) {
		t.Fatalf("kötü uç fazla çağrıldı = %d", bad.calls.Load())
	}
}

func TestRouter_BothUnhealthyUsesOldest(t *testing.T) {
	a := &stubClient{err: timeoutErr()}
	b := &stubClient{err: timeoutErr()}
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	r := NewRouter([]NamedClient{
		{Name: EndpointUC1, Client: a},
		{Name: EndpointUC2, Client: b},
	}, nil)
	r.now = func() time.Time { return now }

	for i := 0; i < unhealthyAfter; i++ {
		_, _ = r.Generate(context.Background(), "s", "u")
	}
	// uc1 became unhealthy first (picked while both healthy by name).
	now = now.Add(time.Second)
	r.now = func() time.Time { return now }
	for i := 0; i < unhealthyAfter; i++ {
		_, _ = r.Generate(context.Background(), "s", "u")
	}
	beforeA, beforeB := a.calls.Load(), b.calls.Load()
	_, _ = r.Generate(context.Background(), "s", "u")
	if a.calls.Load() != beforeA+1 {
		t.Fatalf("en eski sağlıksız (uc1) denenmeli: uc1 %d→%d uc2 %d→%d", beforeA, a.calls.Load(), beforeB, b.calls.Load())
	}
}

func TestRouter_ThreeFailuresMarkUnhealthyThenProbe(t *testing.T) {
	c := &stubClient{err: timeoutErr()}
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	r := NewRouter([]NamedClient{{Name: EndpointUC1, Client: c}}, nil)
	r.now = func() time.Time { return now }
	r.cooldown = time.Minute

	for i := 0; i < unhealthyAfter; i++ {
		_, _ = r.Generate(context.Background(), "s", "u")
	}
	r.mu.Lock()
	if !r.eps[0].unhealthy {
		r.mu.Unlock()
		t.Fatal("3 hatadan sonra sağlıksız olmalı")
	}
	until := r.eps[0].unhealthyUntil
	r.mu.Unlock()
	if !until.Equal(now.Add(time.Minute)) {
		t.Fatalf("sağlıksızlık bitiş = %s", until)
	}

	c.err = nil
	now = now.Add(61 * time.Second)
	r.now = func() time.Time { return now }
	out, err := r.Generate(context.Background(), "s", "u")
	if err != nil || out != "ok" {
		t.Fatalf("sınama başarılı olmalı: %v %s", err, out)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.eps[0].unhealthy {
		t.Fatal("başarılı sınama sonrası sağlıklı olmalı")
	}
}

func TestRouter_503DoesNotMarkUnhealthy(t *testing.T) {
	c := &stubClient{err: coldErr()}
	r := NewRouter([]NamedClient{{Name: EndpointUC1, Client: c}}, nil)
	for i := 0; i < 6; i++ {
		_, err := r.Generate(context.Background(), "s", "u")
		if !errors.Is(err, ErrColdStart) {
			t.Fatalf("err = %v", err)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.eps[0].unhealthy {
		t.Fatal("503 sağlıksızlık saymamalı")
	}
	if r.eps[0].consecutiveFail != 0 {
		t.Fatalf("ardışık hata = %d", r.eps[0].consecutiveFail)
	}
}

func TestRouter_SingleEndpoint(t *testing.T) {
	c := &stubClient{}
	r := NewRouter([]NamedClient{{Name: EndpointUC1, Client: c}}, nil)
	out, err := r.Generate(context.Background(), "s", "u")
	if err != nil || out != "ok" {
		t.Fatalf("%v %s", err, out)
	}
	if c.calls.Load() != 1 {
		t.Fatalf("çağrı = %d", c.calls.Load())
	}
}
