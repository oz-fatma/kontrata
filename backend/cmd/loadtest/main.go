package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	api := flag.String("api", envOr("LOADTEST_API", "http://localhost:8080"), "API kökü")
	sozlesme := flag.String("sozlesme", "", "PDF yolu")
	eszamanli := flag.Int("eszamanli", 5, "eşzamanlı yükleme")
	tekrar := flag.Int("tekrar", 2, "tur sayısı")
	eposta := flag.String("eposta", os.Getenv("LOADTEST_EPOSTA"), "giriş e-postası")
	sifre := flag.String("sifre", os.Getenv("LOADTEST_SIFRE"), "giriş şifresi")
	giris := flag.Bool("giris", false, "yalnızca girisYap; geçici jetonu stdout'a yazar")
	gecici := flag.String("gecici-token", os.Getenv("LOADTEST_GECICI_TOKEN"), "mfaDogrula geçici jetonu")
	mfa := flag.String("mfa-kod", os.Getenv("LOADTEST_MFA"), "MFA kodu")
	erisim := flag.String("erisim-jetonu", os.Getenv("LOADTEST_ERISIM_JETONU"), "hazır erişim jetonu; giriş atlanır")
	flag.Parse()

	c := newGQLClient(*api)
	if *giris {
		return runGiris(c, *eposta, *sifre)
	}
	if err := c.authenticate(*erisim, *gecici, *mfa); err != nil {
		return err
	}
	if strings.TrimSpace(*sozlesme) == "" {
		return fmt.Errorf("--sozlesme gerekli")
	}
	if *eszamanli < 1 {
		*eszamanli = 1
	}
	if *tekrar < 1 {
		*tekrar = 1
	}

	pdf, err := os.ReadFile(*sozlesme)
	if err != nil {
		return fmt.Errorf("pdf okunamadı: %w", err)
	}

	start := time.Now().UTC()
	total := *eszamanli * *tekrar
	var mu sync.Mutex
	var uploadErr error
	durs := make([]time.Duration, 0, total)
	var fail int
	sem := make(chan struct{}, *eszamanli)
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			t0 := time.Now()
			id, err := c.upload(pdf, filepath.Base(*sozlesme))
			if err != nil {
				mu.Lock()
				fail++
				if uploadErr == nil {
					uploadErr = err
				}
				mu.Unlock()
				return
			}
			durum, err := c.waitDone(id, 15*time.Minute)
			elapsed := time.Since(t0)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				fail++
				if uploadErr == nil {
					uploadErr = err
				}
				return
			}
			durs = append(durs, elapsed)
			if durum == "HATA" {
				fail++
			}
		}()
	}
	wg.Wait()
	if uploadErr != nil && len(durs) == 0 {
		return uploadErr
	}
	wall := time.Since(start)
	// İzleme kaydı arka planda yazılır; metrik sorgusundan önce bitsin.
	time.Sleep(time.Second)

	met, calls, err := c.metrics(start)
	if err != nil {
		return err
	}

	body := renderReport(*sozlesme, *eszamanli, *tekrar, total, fail, wall, durs, met, calls)
	fmt.Print(body)

	outPath := filepath.Join(docsDir(), fmt.Sprintf("yuk-testi-%s.md", time.Now().UTC().Format("2006-01-02")))
	if err := os.WriteFile(outPath, []byte(body), 0o644); err != nil {
		return fmt.Errorf("rapor yazılamadı: %w", err)
	}
	fmt.Fprintf(os.Stderr, "rapor yazıldı %s\n", outPath)
	return nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func docsDir() string {
	for _, p := range []string{"docs", "../docs"} {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return "docs"
}

type gqlClient struct {
	url, token, deviceID string
	http                 *http.Client
}

func newGQLClient(api string) *gqlClient {
	return &gqlClient{
		url:      strings.TrimRight(api, "/") + "/graphql",
		deviceID: "loadtest-cihaz",
		http:     &http.Client{Timeout: 2 * time.Minute},
	}
}

func runGiris(c *gqlClient, eposta, sifre string) error {
	if strings.TrimSpace(eposta) == "" || strings.TrimSpace(sifre) == "" {
		return fmt.Errorf("--eposta ve --sifre (veya LOADTEST_EPOSTA / LOADTEST_SIFRE) gerekli")
	}
	token, err := c.girisYap(eposta, sifre)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "geçici jeton yazıldı; MFA kodunu API günlüğünden alın")
	fmt.Println(token)
	return nil
}

func (c *gqlClient) authenticate(erisim, gecici, mfaKod string) error {
	if t := strings.TrimSpace(erisim); t != "" {
		c.token = t
		return nil
	}
	gecici = strings.TrimSpace(gecici)
	mfaKod = strings.TrimSpace(mfaKod)
	if gecici == "" || mfaKod == "" {
		return fmt.Errorf("oturum için --erisim-jetonu veya --gecici-token ve --mfa-kod gerekli (önce make loadtest-giris)")
	}
	return c.mfaDogrula(gecici, mfaKod)
}

func (c *gqlClient) girisYap(eposta, sifre string) (string, error) {
	var giris struct {
		GirisYap struct {
			MfaGerekli  bool
			GeciciToken string
		}
	}
	if err := c.post(`mutation ($e: String!, $s: String!) { girisYap(eposta: $e, sifre: $s) { mfaGerekli geciciToken } }`,
		map[string]any{"e": eposta, "s": sifre}, &giris); err != nil {
		return "", err
	}
	if !giris.GirisYap.MfaGerekli || giris.GirisYap.GeciciToken == "" {
		return "", fmt.Errorf("beklenmeyen giriş yanıtı")
	}
	return giris.GirisYap.GeciciToken, nil
}

func (c *gqlClient) mfaDogrula(gecici, kod string) error {
	var oturum struct {
		MfaDogrula struct {
			ErisimJetonu string
		}
	}
	if err := c.post(`mutation ($t: String!, $k: String!) { mfaDogrula(geciciToken: $t, kod: $k) { erisimJetonu yenilemeJetonu } }`,
		map[string]any{"t": gecici, "k": kod}, &oturum); err != nil {
		return err
	}
	c.token = oturum.MfaDogrula.ErisimJetonu
	if c.token == "" {
		return fmt.Errorf("oturum açılamadı")
	}
	return nil
}

func (c *gqlClient) upload(pdf []byte, name string) (string, error) {
	query := `mutation ($dosya: Upload!) { sozlesmeYukle(dosya: $dosya) { id durum } }`
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	ops, _ := json.Marshal(map[string]any{"query": query, "variables": map[string]any{"dosya": nil}})
	_ = w.WriteField("operations", string(ops))
	_ = w.WriteField("map", `{"0":["variables.dosya"]}`)
	part, err := w.CreateFormFile("0", name)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(pdf); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, c.url, &buf)
	if err != nil {
		return "", err
	}
	c.headers(req)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var wrap struct {
		Data struct {
			SozlesmeYukle struct {
				ID string
			}
		}
		Errors []struct{ Message string }
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return "", fmt.Errorf("yükleme yanıtı okunamadı")
	}
	if len(wrap.Errors) > 0 {
		return "", fmt.Errorf("%s", wrap.Errors[0].Message)
	}
	if wrap.Data.SozlesmeYukle.ID == "" {
		return "", fmt.Errorf("yükleme kimliği yok")
	}
	return wrap.Data.SozlesmeYukle.ID, nil
}

func (c *gqlClient) waitDone(id string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var out struct {
			Sozlesme *struct{ Durum string }
		}
		if err := c.post(`query ($id: ID!) { sozlesme(id: $id) { durum } }`, map[string]any{"id": id}, &out); err != nil {
			return "", err
		}
		if out.Sozlesme == nil {
			return "", fmt.Errorf("sözleşme yok")
		}
		d := out.Sozlesme.Durum
		if d != "YUKLENDI" && d != "ISLENIYOR" {
			return d, nil
		}
		time.Sleep(2 * time.Second)
	}
	return "", fmt.Errorf("zaman aşımı")
}

type metrikDTO struct {
	ToplamCagri    int32
	BasariliCagri  int32
	BasarisizCagri int32
	OrtalamaSureMs float64
	P95SureMs      float64
	UcBazinda      []struct {
		UcAdi          string
		Cagri          int32
		OrtalamaSureMs float64
		BasariOrani    float64
	}
	HataDagilimi []struct {
		HataTipi string
		Adet     int32
	}
}

func (c *gqlClient) metrics(since time.Time) (metrikDTO, []struct {
	UcAdi    string
	SureMs   int32
	Basarili bool
	HataTipi string
}, error) {
	var out struct {
		LlmMetrikleri metrikDTO
		LlmCagrilari  []struct {
			UcAdi    string
			SureMs   int32
			Basarili bool
			HataTipi string
		}
	}
	q := `query ($t: Time!) {
  llmMetrikleri(baslangic: $t) {
    toplamCagri basariliCagri basarisizCagri ortalamaSureMs p95SureMs
    ucBazinda { ucAdi cagri ortalamaSureMs basariOrani }
    hataDagilimi { hataTipi adet }
  }
  llmCagrilari(limit: 100) { ucAdi sureMs basarili hataTipi }
}`
	if err := c.post(q, map[string]any{"t": since.UTC().Format(time.RFC3339Nano)}, &out); err != nil {
		return metrikDTO{}, nil, err
	}
	return out.LlmMetrikleri, out.LlmCagrilari, nil
}

func (c *gqlClient) post(query string, vars map[string]any, dest any) error {
	payload := map[string]any{"query": query}
	if vars != nil {
		payload["variables"] = vars
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.headers(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var wrap struct {
		Data   json.RawMessage
		Errors []struct{ Message string }
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return fmt.Errorf("graphql yanıtı okunamadı")
	}
	if len(wrap.Errors) > 0 {
		return fmt.Errorf("%s", wrap.Errors[0].Message)
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(wrap.Data, dest)
}

func (c *gqlClient) headers(req *http.Request) {
	req.Header.Set("X-Device-Id", c.deviceID)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func renderReport(pdf string, conc, tekrar, total, fail int, wall time.Duration, durs []time.Duration, met metrikDTO, _ []struct {
	UcAdi    string
	SureMs   int32
	Basarili bool
	HataTipi string
}) string {
	var avg, p95 time.Duration
	if len(durs) > 0 {
		var sum time.Duration
		cp := append([]time.Duration(nil), durs...)
		sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
		for _, d := range cp {
			sum += d
		}
		avg = sum / time.Duration(len(cp))
		rank := int(math.Ceil(0.95 * float64(len(cp))))
		if rank < 1 {
			rank = 1
		}
		if rank > len(cp) {
			rank = len(cp)
		}
		p95 = cp[rank-1]
	}
	ok := total - fail
	oran := 0.0
	if total > 0 {
		oran = float64(ok) / float64(total)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Yük testi %s\n\n", time.Now().UTC().Format("2006-01-02"))
	fmt.Fprintf(&b, "- PDF: `%s`\n", pdf)
	fmt.Fprintf(&b, "- Eşzamanlı: %d, tekrar: %d, toplam: %d\n", conc, tekrar, total)
	fmt.Fprintf(&b, "- LLM metrikleri bu koşunun başlangıcından itibaren (önceki koşular dahil değil)\n\n")
	fmt.Fprintf(&b, "## Süre\n\n")
	fmt.Fprintf(&b, "| Ölçüt | Değer |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| Toplam süre | %s |\n", wall.Round(time.Millisecond))
	fmt.Fprintf(&b, "| İstek başına ortalama | %s |\n", avg.Round(time.Millisecond))
	fmt.Fprintf(&b, "| p95 | %s |\n\n", p95.Round(time.Millisecond))
	fmt.Fprintf(&b, "## Sonuç\n\n")
	fmt.Fprintf(&b, "- Başarı oranı: %.0f%% (%d/%d)\n\n", oran*100, ok, total)
	fmt.Fprintf(&b, "## Uç dağılımı\n\n")
	if len(met.UcBazinda) == 0 {
		b.WriteString("Kayıt yok.\n\n")
	} else {
		b.WriteString("| Uç | Çağrı | Ort. süre (ms) | Başarı |\n| --- | --- | --- | --- |\n")
		for _, u := range met.UcBazinda {
			fmt.Fprintf(&b, "| %s | %d | %.0f | %.0f%% |\n", u.UcAdi, u.Cagri, u.OrtalamaSureMs, u.BasariOrani*100)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "## Hata dağılımı\n\n")
	if len(met.HataDagilimi) == 0 {
		b.WriteString("Hata yok.\n")
	} else {
		b.WriteString("| Tip | Adet |\n| --- | --- |\n")
		for _, h := range met.HataDagilimi {
			fmt.Fprintf(&b, "| %s | %d |\n", h.HataTipi, h.Adet)
		}
	}
	return b.String()
}
