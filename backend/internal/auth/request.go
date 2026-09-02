package auth

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type metaKey struct{}

// RequestMeta denetim kaydı için istek kökenidir. E-posta veya gövde içermez.
type RequestMeta struct {
	IP             string
	UserAgent      string
	AcceptLanguage string
	DeviceID       string
}

// WithRequestMeta meta bilgilerini bağlama yazar.
func WithRequestMeta(ctx context.Context, meta RequestMeta) context.Context {
	return context.WithValue(ctx, metaKey{}, meta)
}

// MetaFrom bağlamdaki istek bilgilerini okur; yoksa sıfır değer döner.
func MetaFrom(ctx context.Context) RequestMeta {
	meta, _ := ctx.Value(metaKey{}).(RequestMeta)
	return meta
}

// RequestMiddleware Chi uyumlu sarmalayıcıdır; IP ve kullanıcı ajanını bağlama yazar.
func RequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := RequestMeta{
			IP:             ClientIP(r),
			UserAgent:      r.UserAgent(),
			AcceptLanguage: strings.TrimSpace(r.Header.Get("Accept-Language")),
			DeviceID:       strings.TrimSpace(r.Header.Get("X-Device-Id")),
		}
		next.ServeHTTP(w, r.WithContext(WithRequestMeta(r.Context(), meta)))
	})
}

// ClientIP X-Forwarded-For'un ilk değerini, yoksa RemoteAddr'i döner.
func ClientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		first = strings.TrimSpace(first)
		if first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
