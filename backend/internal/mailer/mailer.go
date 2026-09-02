package mailer

import "strings"

// Mailer e-posta gönderir. Konu ve gövde dışında alıcı loglanmamalıdır.
type Mailer interface {
	Gonder(alici, konu, govde string) error
}

// MaskEposta alıcıyı günlük için gizler. yerel@alan → l***@alan
func MaskEposta(eposta string) string {
	eposta = strings.TrimSpace(eposta)
	at := strings.LastIndex(eposta, "@")
	if at <= 0 || at == len(eposta)-1 {
		return "***"
	}
	local, domain := eposta[:at], eposta[at+1:]
	r := []rune(local)
	if len(r) == 0 {
		return "***@" + domain
	}
	return string(r[0]) + "***@" + domain
}
