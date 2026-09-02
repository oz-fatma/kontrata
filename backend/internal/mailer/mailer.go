package mailer

import "strings"

// Mailer e-posta gönderir. Konu ve gövde dışında alıcı loglanmamalıdır.
type Mailer interface {
	Send(to, subject, body string) error
}

// MaskEmail alıcıyı günlük için gizler. yerel@alan → l***@alan
func MaskEmail(email string) string {
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "***"
	}
	local, domain := email[:at], email[at+1:]
	r := []rune(local)
	if len(r) == 0 {
		return "***@" + domain
	}
	return string(r[0]) + "***@" + domain
}
