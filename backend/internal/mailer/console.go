package mailer

import "log"

// ConsoleMailer geliştirme için iletiyi standart loga yazar; alıcıyı maskeler.
type ConsoleMailer struct{}

// NewConsole konsol göndericisi döner.
func NewConsole() *ConsoleMailer {
	return &ConsoleMailer{}
}

// Gonder alıcıyı maskeleyerek konu ve gövdeyi basar.
func (m *ConsoleMailer) Gonder(alici, konu, govde string) error {
	log.Printf("posta gönderildi alici=%s konu=%s\n%s", MaskEposta(alici), konu, govde)
	return nil
}
