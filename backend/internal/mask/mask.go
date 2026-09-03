// Package mask sözleşme metnindeki kişisel veriyi LLM çağrısından önce örter.
// Yönetici kapatamaz; her model isteğinden önce çalışır.
package mask

import (
	"regexp"
)

const (
	emailToken = "[EPOSTA]"
	phoneToken = "[TELEFON]"
	tcknToken  = "[TCKN]"
)

var (
	emailRe = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	// +90 sonrası 10 hane (cep 5xx veya saha 2xx). Ayırıcıda satır sonu da var;
	// PDF metin çıkarımı numarayı satır ortasında bölebiliyor.
	phonePlusRe = regexp.MustCompile(`\+90(?:[\s./\-]*\d){10}`)
	// 05xx xxx xx xx (ulusal cep); satır sonu ayırıcı sayılır.
	phoneLocalRe = regexp.MustCompile(`(^|[^\d])(05(?:[\s./\-]*\d){9})`)
	tcknRe       = regexp.MustCompile(`\b\d{11}\b`)
)

// Result örtülen metin ve kaç desenin değiştiğidir. İçerik loglanmaz.
type Result struct {
	Text  string
	Count int
}

// Apply e-posta, telefon ve 11 haneli sayı (TCKN) desenlerini değiştirir.
func Apply(text string) Result {
	if text == "" {
		return Result{Text: text}
	}
	n := 0
	out := emailRe.ReplaceAllStringFunc(text, func(string) string {
		n++
		return emailToken
	})
	out = phonePlusRe.ReplaceAllStringFunc(out, func(string) string {
		n++
		return phoneToken
	})
	out = phoneLocalRe.ReplaceAllStringFunc(out, func(m string) string {
		parts := phoneLocalRe.FindStringSubmatch(m)
		if len(parts) < 3 {
			return m
		}
		n++
		return parts[1] + phoneToken
	})
	out = tcknRe.ReplaceAllStringFunc(out, func(string) string {
		n++
		return tcknToken
	})
	return Result{Text: out, Count: n}
}
