package mailer

import (
	"net/url"
	"os"
	"strings"
	"sync"
)

const defaultAppURL = "http://localhost:3000"

var (
	appURLMu sync.RWMutex
	appURL   = defaultAppURL
)

// SetAppURL web arayüzünün temel adresini ayarlar (sondaki / atılır).
func SetAppURL(raw string) {
	appURLMu.Lock()
	defer appURLMu.Unlock()
	raw = strings.TrimSpace(raw)
	if raw == "" {
		appURL = defaultAppURL
		return
	}
	appURL = strings.TrimRight(raw, "/")
}

// AppURL yapılandırılmış temel adresi döner.
func AppURL() string {
	appURLMu.RLock()
	defer appURLMu.RUnlock()
	return appURL
}

// AppURLFromEnv APP_URL ortam değişkenini okur; boşsa varsayılanı kullanır.
func AppURLFromEnv() string {
	if u := strings.TrimSpace(os.Getenv("APP_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultAppURL
}

// TokenLink sayfa yolu ve token ile tam bağlantı üretir.
func TokenLink(path, token string) string {
	base := AppURL()
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u, err := url.Parse(base + path)
	if err != nil {
		return base + path + "?token=" + url.QueryEscape(token)
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String()
}
