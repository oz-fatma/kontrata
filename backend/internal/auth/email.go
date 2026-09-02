package auth

import (
	"errors"
	"strings"
	"unicode"
)

// ErrInvalidEmail kullanıcıya dönebilen biçim hatasıdır.
var ErrInvalidEmail = errors.New("e-posta adresi geçersiz")

// NormalizeEposta boşlukları atar ve küçük harfe çevirir.
func NormalizeEposta(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	at := strings.LastIndex(s, "@")
	if at < 1 || at == len(s)-1 {
		return "", ErrInvalidEmail
	}
	local, domain := s[:at], s[at+1:]
	if local == "" || domain == "" || !strings.Contains(domain, ".") {
		return "", ErrInvalidEmail
	}
	for _, r := range s {
		if unicode.IsSpace(r) {
			return "", ErrInvalidEmail
		}
	}
	return s, nil
}
