package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/mailer"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const (
	kayitMesaji        = "Kayıt alındı. E-posta adresinize doğrulama iletisi gönderildi."
	dogrulamaKonu      = "E-posta adresinizi doğrulayın"
	sifreSifirlamaKonu = "Şifre sıfırlama"
	defaultResendEvery = 5 * time.Minute
)

// AuthService kayıt ve e-posta doğrulama iş kurallarını taşır.
type AuthService struct {
	users        *repository.KullaniciRepository
	tokens       *repository.DogrulamaTokenRepository
	audit        *repository.DenetimRepository
	mailer       mailer.Mailer
	params       auth.Params
	limiter      *resendLimiter
	resetLimiter *resendLimiter
}

func NewAuthService(
	users *repository.KullaniciRepository,
	tokens *repository.DogrulamaTokenRepository,
	audit *repository.DenetimRepository,
	m mailer.Mailer,
	params auth.Params,
) *AuthService {
	if params.Time == 0 {
		params = auth.DefaultParams()
	}
	if m == nil {
		m = mailer.NewConsole()
	}
	return &AuthService{
		users:        users,
		tokens:       tokens,
		audit:        audit,
		mailer:       m,
		params:       params,
		limiter:      newResendLimiter(defaultResendEvery),
		resetLimiter: newResendLimiter(defaultResendEvery),
	}
}

// KayitOl yeni hesap açar veya mevcut e-posta için aynı yanıtı döner.
// Kullanıcı nesnesi ve token dönülmez.
func (s *AuthService) KayitOl(ctx context.Context, eposta, sifre string) (*model.KayitSonucu, error) {
	sonuc := &model.KayitSonucu{Basarili: true, Mesaj: kayitMesaji}

	norm, err := auth.NormalizeEposta(eposta)
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(sifre, s.params)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user := repository.Kullanici{
		Eposta:           norm,
		SifreHash:        hash,
		EpostaDogrulandi: false,
		Durum:            repository.DurumBeklemede,
		OlusturmaTarihi:  now,
		GuncellemeTarihi: now,
	}
	if err := s.users.Create(ctx, &user); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			existing, getErr := s.users.GetByEposta(ctx, norm)
			if getErr == nil {
				s.writeAudit(ctx, &existing.ID, repository.OlayKayit, "zaten_kayitli")
			} else {
				s.writeAudit(ctx, nil, repository.OlayKayit, "zaten_kayitli")
			}
			return sonuc, nil
		}
		log.Printf("kullanıcı kaydı başarısız")
		return nil, err
	}
	if err := s.issueVerification(ctx, &user); err != nil {
		log.Printf("doğrulama kodu hazırlanamadı")
	}
	s.writeAudit(ctx, &user.ID, repository.OlayKayit, "")
	return sonuc, nil
}

// EpostaDogrula düz metin kodu doğrular. Süresi dolmuş veya kullanılmış kod false döner.
func (s *AuthService) EpostaDogrula(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	doc, err := s.tokens.Consume(ctx, auth.HashToken(token), repository.AmacEpostaDogrulama, time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		log.Printf("doğrulama kodu okunamadı")
		return false, err
	}
	if err := s.users.MarkEmailVerified(ctx, doc.KullaniciID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		log.Printf("e-posta doğrulama işlenemedi")
		return false, err
	}
	s.writeAudit(ctx, &doc.KullaniciID, repository.OlayEpostaDogrulandi, "")
	return true, nil
}

// DogrulamaTekrarGonder e-posta kayıtlı olsun ya da olmasın true döner.
func (s *AuthService) DogrulamaTekrarGonder(ctx context.Context, eposta string) (bool, error) {
	norm, err := auth.NormalizeEposta(eposta)
	if err != nil {
		return true, nil
	}
	user, err := s.users.GetByEposta(ctx, norm)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		log.Printf("doğrulama tekrarı başarısız")
		return false, err
	}
	if !s.limiter.allow(norm) {
		var id *bson.ObjectID
		if user != nil {
			id = &user.ID
		}
		s.writeAudit(ctx, id, repository.OlayDogrulamaTekrarGonderildi, "hiz_siniri")
		return true, nil
	}
	if user == nil {
		s.writeAudit(ctx, nil, repository.OlayDogrulamaTekrarGonderildi, "kullanici_yok")
		return true, nil
	}
	if user.EpostaDogrulandi {
		s.writeAudit(ctx, &user.ID, repository.OlayDogrulamaTekrarGonderildi, "zaten_dogrulanmis")
		return true, nil
	}
	if err := s.issueVerification(ctx, user); err != nil {
		log.Printf("doğrulama kodu hazırlanamadı")
	}
	s.writeAudit(ctx, &user.ID, repository.OlayDogrulamaTekrarGonderildi, "")
	return true, nil
}

// SifreSifirlamaIste e-posta kayıtlı olsun ya da olmasın true döner.
func (s *AuthService) SifreSifirlamaIste(ctx context.Context, eposta string) (bool, error) {
	norm, err := auth.NormalizeEposta(eposta)
	if err != nil {
		return true, nil
	}
	user, err := s.users.GetByEposta(ctx, norm)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		log.Printf("şifre sıfırlama isteği başarısız")
		return false, err
	}
	if !s.resetLimiter.allow(norm) {
		var id *bson.ObjectID
		if user != nil {
			id = &user.ID
		}
		s.writeAudit(ctx, id, repository.OlaySifreSifirlamaIstendi, "hiz_siniri")
		return true, nil
	}
	if user == nil {
		s.writeAudit(ctx, nil, repository.OlaySifreSifirlamaIstendi, "kullanici_yok")
		return true, nil
	}
	if err := s.issuePasswordReset(ctx, user); err != nil {
		log.Printf("şifre sıfırlama kodu hazırlanamadı")
	}
	s.writeAudit(ctx, &user.ID, repository.OlaySifreSifirlamaIstendi, "")
	return true, nil
}

// SifreSifirla tek kullanımlık kodla yeni şifre yazar. Zayıf şifre hata döner.
func (s *AuthService) SifreSifirla(ctx context.Context, token, yeniSifre string) (bool, error) {
	hash, err := auth.HashPassword(yeniSifre, s.params)
	if err != nil {
		return false, err
	}
	if token == "" {
		return false, nil
	}
	doc, err := s.tokens.Consume(ctx, auth.HashToken(token), repository.AmacSifreSifirlama, time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		log.Printf("şifre sıfırlama kodu okunamadı")
		return false, err
	}
	if err := s.users.UpdatePassword(ctx, doc.KullaniciID, hash); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		log.Printf("şifre güncellenemedi")
		return false, err
	}
	s.revokeSessions(ctx, doc.KullaniciID)
	s.writeAudit(ctx, &doc.KullaniciID, repository.OlaySifreSifirlandi, "")
	return true, nil
}

func (s *AuthService) issuePasswordReset(ctx context.Context, user *repository.Kullanici) error {
	if err := s.tokens.InvalidateUnused(ctx, user.ID, repository.AmacSifreSifirlama); err != nil {
		return err
	}
	plain, hash, err := auth.NewToken()
	if err != nil {
		return err
	}
	doc := repository.DogrulamaTokeni{
		KullaniciID: user.ID,
		Token:       hash,
		Amac:        repository.AmacSifreSifirlama,
		SonKullanma: time.Now().UTC().Add(auth.PasswordResetTTL),
		Kullanildi:  false,
	}
	if err := s.tokens.Create(ctx, &doc); err != nil {
		return err
	}
	if err := s.mailer.Gonder(user.Eposta, sifreSifirlamaKonu, sifreSifirlamaGovde(plain)); err != nil {
		log.Printf("şifre sıfırlama iletisi gönderilemedi")
	}
	return nil
}

func sifreSifirlamaGovde(token string) string {
	return fmt.Sprintf("Kontrata şifre sıfırlama\n\nSıfırlama kodunuz:\n\n%s\n\nBu kod 1 saat geçerlidir.\n", token)
}

// revokeSessions oturum koleksiyonu gelince tüm aktif oturumları düşürür.
func (s *AuthService) revokeSessions(ctx context.Context, kullaniciID bson.ObjectID) {
	// TODO: oturum koleksiyonu Prompt 3b'de; bu kullanıcının tüm aktif oturumlarını iptal et.
	_ = ctx
	_ = kullaniciID
}

func (s *AuthService) issueVerification(ctx context.Context, user *repository.Kullanici) error {
	if err := s.tokens.InvalidateUnused(ctx, user.ID, repository.AmacEpostaDogrulama); err != nil {
		return err
	}
	plain, hash, err := auth.NewToken()
	if err != nil {
		return err
	}
	doc := repository.DogrulamaTokeni{
		KullaniciID: user.ID,
		Token:       hash,
		Amac:        repository.AmacEpostaDogrulama,
		SonKullanma: time.Now().UTC().Add(auth.TokenTTL),
		Kullanildi:  false,
	}
	if err := s.tokens.Create(ctx, &doc); err != nil {
		return err
	}
	if err := s.mailer.Gonder(user.Eposta, dogrulamaKonu, dogrulamaGovde(plain)); err != nil {
		log.Printf("doğrulama iletisi gönderilemedi")
	}
	return nil
}

func dogrulamaGovde(token string) string {
	return fmt.Sprintf("Kontrata e-posta doğrulama\n\nDoğrulama kodunuz:\n\n%s\n\nBu kod 24 saat geçerlidir.\n", token)
}

func (s *AuthService) writeAudit(ctx context.Context, kullaniciID *bson.ObjectID, olay, detay string) {
	meta := auth.MetaFrom(ctx)
	rec := repository.DenetimKaydi{
		KullaniciID:    kullaniciID,
		Olay:           olay,
		IPAdresi:       meta.IP,
		KullaniciAjani: meta.UserAgent,
		Zaman:          time.Now().UTC(),
		Detay:          detay,
	}
	if err := s.audit.Insert(ctx, &rec); err != nil {
		log.Printf("denetim kaydı yazılamadı")
	}
}

type resendLimiter struct {
	mu    sync.Mutex
	last  map[string]time.Time
	every time.Duration
}

func newResendLimiter(every time.Duration) *resendLimiter {
	return &resendLimiter{last: make(map[string]time.Time), every: every}
}

func (l *resendLimiter) allow(epostaNorm string) bool {
	key := epostaKey(epostaNorm)
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if prev, ok := l.last[key]; ok && now.Sub(prev) < l.every {
		return false
	}
	l.last[key] = now
	return true
}

func epostaKey(eposta string) string {
	sum := sha256.Sum256([]byte(eposta))
	return hex.EncodeToString(sum[:])
}
