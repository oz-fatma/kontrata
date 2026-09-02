package mailer

import (
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
)

var errSend = errors.New("e-posta gönderilemedi")

// SMTPConfig SMTP gönderimi için bağlantı bilgisi.
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// SMTPMailer net/smtp ile gönderir.
type SMTPMailer struct {
	cfg SMTPConfig
}

// NewSMTP SMTP göndericisi döner.
func NewSMTP(cfg SMTPConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

// New kind=smtp ise SMTP, aksi halde konsol göndericisi seçer.
func New(kind string, cfg SMTPConfig) Mailer {
	if strings.EqualFold(strings.TrimSpace(kind), "smtp") {
		return NewSMTP(cfg)
	}
	return NewConsole()
}

// Gonder SMTP üzerinden düz metin ileti yollar. Hata ayrıntısı (alıcı içerebilir) loglanmaz.
func (m *SMTPMailer) Gonder(alici, konu, govde string) error {
	from := strings.TrimSpace(m.cfg.From)
	if from == "" || alici == "" {
		return errSend
	}
	port := m.cfg.Port
	if port == 0 {
		port = 587
	}
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(port))
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, alici, mime.QEncoding.Encode("UTF-8", konu), govde,
	))
	var auth smtp.Auth
	if m.cfg.User != "" {
		auth = smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
	}
	if err := smtp.SendMail(addr, auth, from, []string{alici}, msg); err != nil {
		log.Printf("smtp gönderimi başarısız")
		return errSend
	}
	return nil
}
