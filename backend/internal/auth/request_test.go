package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPPrefersXForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	r.RemoteAddr = "192.0.2.10:9999"
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	if got := ClientIP(r); got != "203.0.113.9" {
		t.Fatalf("got %q", got)
	}
}

func TestClientIPRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	r.RemoteAddr = "192.0.2.10:9999"
	if got := ClientIP(r); got != "192.0.2.10" {
		t.Fatalf("got %q", got)
	}
}

func TestRequestMiddlewareWritesContext(t *testing.T) {
	var got RequestMeta
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = MetaFrom(r.Context())
	})
	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.RemoteAddr = "192.0.2.10:1"
	req.Header.Set("User-Agent", "kontrata-test")
	req.Header.Set("X-Forwarded-For", "198.51.100.7")
	RequestMiddleware(inner).ServeHTTP(httptest.NewRecorder(), req)
	if got.IP != "198.51.100.7" {
		t.Fatalf("IP = %q", got.IP)
	}
	if got.UserAgent != "kontrata-test" {
		t.Fatalf("UserAgent = %q", got.UserAgent)
	}
}

func TestRequestMiddlewareDeviceHeaders(t *testing.T) {
	var got RequestMeta
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = MetaFrom(r.Context())
	})
	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.Header.Set("User-Agent", "Electron")
	req.Header.Set("Accept-Language", "tr-TR")
	req.Header.Set("X-Device-Id", "cihaz-1")
	RequestMiddleware(inner).ServeHTTP(httptest.NewRecorder(), req)
	if got.DeviceID != "cihaz-1" || got.AcceptLanguage != "tr-TR" {
		t.Fatalf("meta = %+v", got)
	}
}
