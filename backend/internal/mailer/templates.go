package mailer

import "strings"

func tokenActionBody(title, action, path, codeLabel, token, ttl string) string {
	link := TokenLink(path, token)
	var b strings.Builder
	b.WriteString("Kontrata ")
	b.WriteString(title)
	b.WriteString("\n\nAşağıdaki bağlantıya tıklayarak ")
	b.WriteString(action)
	b.WriteString(":\n")
	b.WriteString(link)
	b.WriteString("\n\nBağlantı açılmazsa aşağıdaki kodu kullanın.\n\n")
	b.WriteString(codeLabel)
	b.WriteString("\n")
	b.WriteString(token)
	b.WriteString("\n\n")
	b.WriteString(ttl)
	b.WriteString("\n")
	return b.String()
}

// VerificationBody e-posta doğrulama iletisinin düz metin gövdesidir.
func VerificationBody(token string) string {
	return tokenActionBody(
		"e-posta doğrulama",
		"e-posta adresinizi doğrulayabilirsiniz",
		"/dogrula",
		"Doğrulama kodunuz:",
		token,
		"Bu kod 24 saat geçerlidir.",
	)
}

// PasswordResetBody şifre sıfırlama iletisinin düz metin gövdesidir.
func PasswordResetBody(token string) string {
	return tokenActionBody(
		"şifre sıfırlama",
		"şifrenizi sıfırlayabilirsiniz",
		"/sifre-sifirla",
		"Sıfırlama kodunuz:",
		token,
		"Bu kod 1 saat geçerlidir.",
	)
}

// AccountDeleteBody hesap silme onayı iletisinin düz metin gövdesidir.
func AccountDeleteBody(token string) string {
	return tokenActionBody(
		"hesap silme",
		"hesap silme işlemini tamamlayabilirsiniz",
		"/ayarlar/",
		"Hesap silme onay kodunuz:",
		token,
		"Bu kod 1 saat geçerlidir.",
	)
}

// InviteBody organizasyon daveti iletisinin düz metin gövdesidir.
func InviteBody(token string) string {
	return tokenActionBody(
		"organizasyon daveti",
		"daveti kabul edebilirsiniz",
		"/kayit",
		"Davet kodunuz:",
		token,
		"Bu kod 7 gün geçerlidir.",
	)
}
