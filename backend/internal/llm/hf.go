package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultTimeout    = 240 * time.Second
	defaultMaxTokens  = 600
	maxColdAttempts   = 8
	maxErrorBodyRunes = 512
)

// ErrUnavailable endpoint yanıt vermediğinde veya zaman aşımında döner.
var ErrUnavailable = errors.New("model yanıt vermedi")

// ErrColdStart soğuk başlangıç denemeleri tükenince döner.
var ErrColdStart = errors.New("endpoint uyanmadı")

var coldStartWaits = []time.Duration{
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	15 * time.Second,
	20 * time.Second,
	20 * time.Second,
	20 * time.Second,
}

type retryKind int

const (
	retryNone retryKind = iota
	retryCold
)

// HFEndpoint HuggingFace Inference Endpoint (kök yol, TGI inputs) istemcisidir.
type HFEndpoint struct {
	URL          string
	Token        string
	HTTP         *http.Client
	MaxNewTokens int
	Timeout      time.Duration
	Sleep        func(ctx context.Context, d time.Duration) error
}

type hfRequest struct {
	Inputs     string       `json:"inputs"`
	Parameters hfParameters `json:"parameters"`
}

type hfParameters struct {
	MaxNewTokens   int      `json:"max_new_tokens"`
	DoSample       bool     `json:"do_sample"`
	Temperature    *float64 `json:"temperature,omitempty"`
	ReturnFullText bool     `json:"return_full_text"`
}

type hfResponse struct {
	GeneratedText string `json:"generated_text"`
}

// NewHFEndpoint CPU endpoint için 240 sn zaman aşımı ve 600 token ile istemci kurar.
// 503 soğuk başlangıçta 8 deneme; zaman aşımı ve 4xx yeniden denenmez.
func NewHFEndpoint(endpointURL, token string, maxTokens int, timeout time.Duration) *HFEndpoint {
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &HFEndpoint{
		URL:          strings.TrimRight(strings.TrimSpace(endpointURL), "/"),
		Token:        token,
		MaxNewTokens: maxTokens,
		Timeout:      timeout,
		HTTP:         &http.Client{Timeout: timeout},
	}
}

// LimitTokens HuggingFace veya yönlendirici istemcisinin token üst sınırını ayarlar.
func LimitTokens(c Client, n int) Client {
	if c == nil || n <= 0 {
		return c
	}
	switch t := c.(type) {
	case *HFEndpoint:
		cp := *t
		cp.MaxNewTokens = n
		return &cp
	case *Router:
		return &limitedRouter{r: t, n: n}
	case *limitedRouter:
		return &limitedRouter{r: t.r, n: n}
	default:
		return c
	}
}

// Generate sohbet şablonunu kurar, endpoint'e gönderir ve üretilen metni döner.
// Prompt ve çıktı loglanmaz; yalnızca gecikme, karakter sayısı ve hata durumu yazılır.
func (c *HFEndpoint) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if c == nil || c.URL == "" {
		return "", ErrUnavailable
	}
	prompt := BuildChatPrompt(systemPrompt, userPrompt)
	inChars := utf8.RuneCountInString(prompt)

	params := hfParameters{
		MaxNewTokens:   c.tokenLimit(),
		DoSample:       false,
		ReturnFullText: false,
	}
	if temp, ok := TemperatureFrom(ctx); ok {
		params.DoSample = true
		params.Temperature = &temp
	}
	body, err := json.Marshal(hfRequest{
		Inputs:     prompt,
		Parameters: params,
	})
	if err != nil {
		log.Printf("llm istek kodlanamadı: %v", err)
		return "", ErrUnavailable
	}

	start := time.Now()
	nCold := 0
	attempt := 0
	var lastKind string
	for {
		attempt++
		t0 := time.Now()
		text, kind, retry, err := c.doOnce(ctx, body)
		lastKind = kind
		outChars := utf8.RuneCountInString(text)
		tipi, cold := hataTipiFromKind(kind)
		emitAttempt(ctx, Attempt{
			Number:    attempt,
			Start:     t0,
			End:       time.Now(),
			InChars:   inChars,
			OutChars:  outChars,
			Success:   err == nil,
			ErrorType: tipi,
			Cold:      cold,
		})
		if err == nil {
			log.Printf("llm tamam gecikme=%s karakter_giris=%d karakter_cikis=%d", time.Since(start), inChars, outChars)
			return text, nil
		}
		if retry != retryCold {
			log.Printf("llm hata gecikme=%s karakter_giris=%d hata=%s", time.Since(start), inChars, kind)
			return "", wrapCallError(kind, ErrUnavailable)
		}
		nCold++
		if nCold >= maxColdAttempts {
			log.Printf("llm endpoint uyanmadı gecikme=%s deneme=%d", time.Since(start), nCold)
			return "", wrapCallError(kind, ErrColdStart)
		}
		wait := coldStartWaits[nCold-1]
		log.Printf("llm endpoint uyanıyor deneme=%d/%d bekleme=%s", nCold, maxColdAttempts, wait)
		if err := c.sleep(ctx, wait); err != nil {
			log.Printf("llm iptal gecikme=%s karakter_giris=%d hata=%s", time.Since(start), inChars, lastKind)
			return "", wrapCallError(lastKind, ErrUnavailable)
		}
	}
}

func (c *HFEndpoint) doOnce(ctx context.Context, body []byte) (string, string, retryKind, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return "", "istek", retryNone, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", classifyNetErr(err), retryNone, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("llm yanıt gövdesi kapatılamadı: %v", err)
		}
	}()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", "yanit_okuma", retryNone, err
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		return "", "503", retryCold, fmt.Errorf("503")
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		log.Printf("llm http_%d %s", resp.StatusCode, errorBodyForLog(raw))
		return "", fmt.Sprintf("http_%d", resp.StatusCode), retryNone, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("llm http_%d %s", resp.StatusCode, errorBodyForLog(raw))
		return "", fmt.Sprintf("http_%d", resp.StatusCode), retryNone, fmt.Errorf("http %d", resp.StatusCode)
	}

	text, err := parseGeneratedText(raw)
	if err != nil {
		return "", "yanit_ayristirma", retryNone, err
	}
	return text, "ok", retryNone, nil
}

func parseGeneratedText(raw []byte) (string, error) {
	var arr []hfResponse
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr[0].GeneratedText, nil
	}
	var obj hfResponse
	if err := json.Unmarshal(raw, &obj); err == nil && obj.GeneratedText != "" {
		return obj.GeneratedText, nil
	}
	return "", errors.New("generated_text yok")
}

func errorBodyForLog(raw []byte) string {
	s := strings.ToValidUTF8(string(bytes.TrimSpace(raw)), "")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "(bos)"
	}
	runes := []rune(s)
	if len(runes) > maxErrorBodyRunes {
		return string(runes[:maxErrorBodyRunes]) + "…"
	}
	return s
}

func classifyNetErr(err error) string {
	if errors.Is(err, context.Canceled) {
		return "iptal"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "zaman_asimi"
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "zaman_asimi"
	}
	var ue *url.Error
	if errors.As(err, &ue) && ue.Timeout() {
		return "zaman_asimi"
	}
	return "ag"
}

func (c *HFEndpoint) tokenLimit() int {
	if c.MaxNewTokens <= 0 {
		return defaultMaxTokens
	}
	return c.MaxNewTokens
}

func (c *HFEndpoint) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &http.Client{Timeout: timeout}
}

func (c *HFEndpoint) sleep(ctx context.Context, d time.Duration) error {
	if fn := c.Sleep; fn != nil {
		return fn(ctx, d)
	}
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
