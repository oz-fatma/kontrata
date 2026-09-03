package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBuildChatPrompt_QwenTemplate(t *testing.T) {
	got := BuildChatPrompt("sistem", "sözleşme")
	want := "<|im_start|>system\nsistem<|im_end|>\n<|im_start|>user\nsözleşme<|im_end|>\n<|im_start|>assistant"
	if got != want {
		t.Fatalf("şablon eşleşmedi\nalınan:\n%q\nbeklenen:\n%q", got, want)
	}
	if strings.HasSuffix(got, "\n") {
		t.Fatal("son satırdan sonra yeni satır olmamalı")
	}
	if !strings.HasSuffix(got, "<|im_start|>assistant") {
		t.Fatal("prompt assistant konumunda bitmeli")
	}
}

func TestHFEndpoint_Success(t *testing.T) {
	var gotInputs string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("yol %s, beklenen /", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization başlığı beklenen değil")
		}
		raw, _ := io.ReadAll(r.Body)
		var req hfRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Errorf("istek JSON değil: %v", err)
		}
		gotInputs = req.Inputs
		if req.Parameters.MaxNewTokens != 600 || req.Parameters.ReturnFullText || req.Parameters.DoSample {
			t.Errorf("parametreler beklenen değil: %+v", req.Parameters)
		}
		if strings.Contains(string(raw), `"temperature"`) {
			t.Errorf("temperature gönderilmemeli: %s", raw)
		}
		if !strings.Contains(string(raw), `"do_sample":false`) && !strings.Contains(string(raw), `"do_sample": false`) {
			t.Errorf("do_sample=false yok: %s", raw)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"generated_text":"{\"donem\":{}}"}]`))
	}))
	t.Cleanup(srv.Close)

	c := testClient(srv.URL)
	out, err := c.Generate(context.Background(), "sys", "user-metin")
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if out != `{"donem":{}}` {
		t.Fatalf("çıktı = %q", out)
	}
	if !strings.Contains(gotInputs, "<|im_start|>system") || !strings.HasSuffix(gotInputs, "<|im_start|>assistant") {
		t.Fatal("gönderilen prompt sohbet şablonu değil")
	}
}

func TestHFEndpoint_RetriesOn503(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := n.Add(1)
		if i < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"generated_text":"ok"}]`))
	}))
	t.Cleanup(srv.Close)

	c := testClient(srv.URL)
	out, err := c.Generate(context.Background(), "s", "u")
	if err != nil {
		t.Fatalf("503 sonrası başarı bekleniyordu: %v", err)
	}
	if out != "ok" {
		t.Fatalf("çıktı = %q", out)
	}
	if n.Load() != 3 {
		t.Fatalf("deneme = %d, beklenen 3", n.Load())
	}
}

func TestHFEndpoint_NoRetryOnTimeout(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		time.Sleep(80 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"generated_text":"geç"}]`))
	}))
	t.Cleanup(srv.Close)

	c := testClient(srv.URL)
	c.HTTP = &http.Client{Timeout: 20 * time.Millisecond}
	_, err := c.Generate(context.Background(), "s", "u")
	if err == nil {
		t.Fatal("zaman aşımında hata bekleniyordu")
	}
	if n.Load() != 1 {
		t.Fatalf("deneme = %d, beklenen 1", n.Load())
	}
}

func TestHFEndpoint_GivesUpAfterRetries(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	c := testClient(srv.URL)
	_, err := c.Generate(context.Background(), "s", "u")
	if !errors.Is(err, ErrColdStart) {
		t.Fatalf("err = %v, beklenen endpoint uyanmadı", err)
	}
	if n.Load() != 8 {
		t.Fatalf("deneme = %d, beklenen 8", n.Load())
	}
}

func TestHFEndpoint_NoRetryOn400(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"temperature (=0.0) has to be a strictly positive float"}`))
	}))
	t.Cleanup(srv.Close)

	var logged bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(prev) })

	c := testClient(srv.URL)
	_, err := c.Generate(context.Background(), "s", "u")
	if err == nil {
		t.Fatal("hata bekleniyordu")
	}
	if n.Load() != 1 {
		t.Fatalf("deneme = %d, beklenen 1", n.Load())
	}
	out := logged.String()
	if !strings.Contains(out, "http_400") || !strings.Contains(out, "strictly positive") {
		t.Fatalf("hata gövdesi loglanmadı: %s", out)
	}
}

func TestHFEndpoint_TemperatureFromContext(t *testing.T) {
	var got hfParameters
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req hfRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Errorf("istek JSON değil: %v", err)
		}
		got = req.Parameters
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"generated_text":"ok"}]`))
	}))
	t.Cleanup(srv.Close)

	c := testClient(srv.URL)
	ctx := WithTemperature(context.Background(), 0.2)
	_, err := c.Generate(ctx, "s", "u")
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if !got.DoSample {
		t.Fatal("do_sample=true bekleniyordu")
	}
	if got.Temperature == nil || *got.Temperature != 0.2 {
		t.Fatalf("temperature = %v", got.Temperature)
	}
}

func testClient(url string) *HFEndpoint {
	return &HFEndpoint{
		URL:          url,
		Token:        "test-token",
		HTTP:         &http.Client{Timeout: 2 * time.Second},
		MaxNewTokens: 600,
		Timeout:      2 * time.Second,
		Sleep:        func(context.Context, time.Duration) error { return nil },
	}
}
