package mailer

import "log"

// ConsoleMailer geliştirme için iletiyi standart loga yazar; alıcıyı maskeler.
type ConsoleMailer struct{}

// NewConsole konsol göndericisi döner.
func NewConsole() *ConsoleMailer {
	return &ConsoleMailer{}
}

// Send alıcıyı maskeleyerek konu ve gövdeyi basar.
func (m *ConsoleMailer) Send(to, subject, body string) error {
	log.Printf("posta gönderildi alici=%s konu=%s\n%s", MaskEmail(to), subject, body)
	return nil
}
