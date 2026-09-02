package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/mailer"
	appmongo "github.com/oz-fatma/kontrata/backend/internal/mongo"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const (
	registerMessage      = "Kayıt alındı. E-posta adresinize doğrulama iletisi gönderildi."
	verificationSubject  = "E-posta adresinizi doğrulayın"
	passwordResetSubject = "Şifre sıfırlama"
	defaultResendEvery   = 5 * time.Minute
)

// AuthService kayıt, doğrulama, MFA ve oturum iş kurallarını taşır.
type AuthService struct {
	users         *repository.UserRepository
	tokens        *repository.VerificationTokenRepository
	mfa           *repository.MFACodeRepository
	sessions      *repository.SessionRepository
	devices       *repository.DeviceRepository
	soz           *repository.ContractRepository
	orgs          *repository.OrganizationRepository
	davets        *repository.InviteRepository
	audit         *repository.AuditRepository
	db            *appmongo.Client
	mailer        mailer.Mailer
	params        auth.Params
	jwt           *auth.JWT
	dummyHash     string
	limiter       *resendLimiter
	resetLimiter  *resendLimiter
	deleteLimiter *resendLimiter
	loginGuard    *loginGuard
	deleteFailAt  string
}

func NewAuthService(
	users *repository.UserRepository,
	tokens *repository.VerificationTokenRepository,
	mfa *repository.MFACodeRepository,
	sessions *repository.SessionRepository,
	devices *repository.DeviceRepository,
	soz *repository.ContractRepository,
	orgs *repository.OrganizationRepository,
	davets *repository.InviteRepository,
	audit *repository.AuditRepository,
	m mailer.Mailer,
	params auth.Params,
	jwtSigner *auth.JWT,
	db *appmongo.Client,
) *AuthService {
	if params.Time == 0 {
		params = auth.DefaultParams()
	}
	if m == nil {
		m = mailer.NewConsole()
	}
	dummy, _ := auth.HashPassword("yer-tutucu-12", params)
	return &AuthService{
		users:         users,
		tokens:        tokens,
		mfa:           mfa,
		sessions:      sessions,
		devices:       devices,
		soz:           soz,
		orgs:          orgs,
		davets:        davets,
		audit:         audit,
		db:            db,
		mailer:        m,
		params:        params,
		jwt:           jwtSigner,
		dummyHash:     dummy,
		limiter:       newResendLimiter(defaultResendEvery),
		resetLimiter:  newResendLimiter(defaultResendEvery),
		deleteLimiter: newResendLimiter(defaultResendEvery),
		loginGuard:    newLoginGuard(),
	}
}

// KayitOl yeni hesap açar veya mevcut e-posta için aynı yanıtı döner.
// Kullanıcı nesnesi ve token dönülmez.
func (s *AuthService) Register(ctx context.Context, eposta, sifre string, hesapTipi *model.HesapTipi, organizasyonAdi *string) (*model.KayitSonucu, error) {
	sonuc := &model.KayitSonucu{Basarili: true, Mesaj: registerMessage}

	tip := repository.AccountIndividual
	if hesapTipi != nil && *hesapTipi != "" {
		tip = string(*hesapTipi)
	}
	orgAd := ""
	if organizasyonAdi != nil {
		orgAd = strings.TrimSpace(*organizasyonAdi)
	}
	if tip == repository.AccountCorporate && orgAd == "" {
		return nil, auth.ErrOrgNameRequired
	}

	norm, err := auth.NormalizeEmail(eposta)
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(sifre, s.params)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user := repository.User{
		Email:         norm,
		PasswordHash:  hash,
		EmailVerified: false,
		Status:        repository.StatusPending,
		AccountType:   tip,
		Role:          repository.RoleOwner,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.users.Create(ctx, &user); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			existing, getErr := s.users.GetByEmail(ctx, norm)
			if getErr == nil {
				s.writeAudit(ctx, &existing.ID, repository.EventRegister, "zaten_kayitli")
			} else {
				s.writeAudit(ctx, nil, repository.EventRegister, "zaten_kayitli")
			}
			return sonuc, nil
		}
		log.Printf("kullanıcı kaydı başarısız: %v", err)
		return nil, err
	}
	if tip == repository.AccountCorporate {
		org := repository.Organization{
			Name:        orgAd,
			OwnerUserID: user.ID,
			Status:      repository.OrgStatusActive,
			CreatedAt:   now,
		}
		if err := s.orgs.Create(ctx, &org); err != nil {
			log.Printf("organizasyon kaydı başarısız: %v", err)
			return nil, err
		}
		if err := s.users.SetOrganization(ctx, user.ID, org.ID, repository.AccountCorporate, repository.RoleOwner); err != nil {
			log.Printf("organizasyon bağlanamadı: %v", err)
			return nil, err
		}
		user.OrganizationID = org.ID
	}
	if err := s.issueVerification(ctx, &user); err != nil {
		log.Printf("doğrulama kodu hazırlanamadı: %v", err)
	}
	s.writeAudit(ctx, &user.ID, repository.EventRegister, "")
	return sonuc, nil
}

// EpostaDogrula düz metin kodu doğrular. Süresi dolmuş veya kullanılmış kod false döner.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	doc, err := s.tokens.Consume(ctx, auth.HashToken(token), repository.PurposeEmailVerification, time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		log.Printf("doğrulama kodu okunamadı: %v", err)
		return false, err
	}
	if err := s.users.MarkEmailVerified(ctx, doc.UserID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		log.Printf("e-posta doğrulama işlenemedi: %v", err)
		return false, err
	}
	s.writeAudit(ctx, &doc.UserID, repository.EventEmailVerified, "")
	return true, nil
}

// DogrulamaTekrarGonder e-posta kayıtlı olsun ya da olmasın true döner.
func (s *AuthService) ResendVerification(ctx context.Context, eposta string) (bool, error) {
	norm, err := auth.NormalizeEmail(eposta)
	if err != nil {
		return true, nil
	}
	user, err := s.users.GetByEmail(ctx, norm)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		log.Printf("doğrulama tekrarı başarısız: %v", err)
		return false, err
	}
	if !s.limiter.allow(norm) {
		var id *bson.ObjectID
		if user != nil {
			id = &user.ID
		}
		s.writeAudit(ctx, id, repository.EventVerificationResent, "hiz_siniri")
		return true, nil
	}
	if user == nil {
		s.writeAudit(ctx, nil, repository.EventVerificationResent, "kullanici_yok")
		return true, nil
	}
	if user.EmailVerified {
		s.writeAudit(ctx, &user.ID, repository.EventVerificationResent, "zaten_dogrulanmis")
		return true, nil
	}
	if err := s.issueVerification(ctx, user); err != nil {
		log.Printf("doğrulama kodu hazırlanamadı: %v", err)
	}
	s.writeAudit(ctx, &user.ID, repository.EventVerificationResent, "")
	return true, nil
}

// SifreSifirlamaIste e-posta kayıtlı olsun ya da olmasın true döner.
func (s *AuthService) RequestPasswordReset(ctx context.Context, eposta string) (bool, error) {
	norm, err := auth.NormalizeEmail(eposta)
	if err != nil {
		return true, nil
	}
	user, err := s.users.GetByEmail(ctx, norm)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		log.Printf("şifre sıfırlama isteği başarısız: %v", err)
		return false, err
	}
	if !s.resetLimiter.allow(norm) {
		var id *bson.ObjectID
		if user != nil {
			id = &user.ID
		}
		s.writeAudit(ctx, id, repository.EventPasswordResetRequested, "hiz_siniri")
		return true, nil
	}
	if user == nil {
		s.writeAudit(ctx, nil, repository.EventPasswordResetRequested, "kullanici_yok")
		return true, nil
	}
	if err := s.issuePasswordReset(ctx, user); err != nil {
		log.Printf("şifre sıfırlama kodu hazırlanamadı: %v", err)
	}
	s.writeAudit(ctx, &user.ID, repository.EventPasswordResetRequested, "")
	return true, nil
}

// SifreSifirla tek kullanımlık kodla yeni şifre yazar. Zayıf şifre hata döner.
func (s *AuthService) ResetPassword(ctx context.Context, token, yeniSifre string) (bool, error) {
	hash, err := auth.HashPassword(yeniSifre, s.params)
	if err != nil {
		return false, err
	}
	if token == "" {
		return false, nil
	}
	doc, err := s.tokens.Consume(ctx, auth.HashToken(token), repository.PurposePasswordReset, time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		log.Printf("şifre sıfırlama kodu okunamadı: %v", err)
		return false, err
	}
	if err := s.users.UpdatePassword(ctx, doc.UserID, hash); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		log.Printf("şifre güncellenemedi: %v", err)
		return false, err
	}
	s.revokeSessions(ctx, doc.UserID)
	s.writeAudit(ctx, &doc.UserID, repository.EventPasswordReset, "")
	return true, nil
}

func (s *AuthService) issuePasswordReset(ctx context.Context, user *repository.User) error {
	if err := s.tokens.InvalidateUnused(ctx, user.ID, repository.PurposePasswordReset); err != nil {
		return err
	}
	plain, hash, err := auth.NewToken()
	if err != nil {
		return err
	}
	doc := repository.VerificationToken{
		UserID:    user.ID,
		Token:     hash,
		Purpose:   repository.PurposePasswordReset,
		ExpiresAt: time.Now().UTC().Add(auth.PasswordResetTTL),
		Used:      false,
	}
	if err := s.tokens.Create(ctx, &doc); err != nil {
		return err
	}
	if err := s.mailer.Send(user.Email, passwordResetSubject, sifreSifirlamaGovde(plain)); err != nil {
		log.Printf("şifre sıfırlama iletisi gönderilemedi: %v", err)
	}
	return nil
}

func sifreSifirlamaGovde(token string) string {
	return fmt.Sprintf("Kontrata şifre sıfırlama\n\nSıfırlama kodunuz:\n\n%s\n\nBu kod 1 saat geçerlidir.\n", token)
}

func (s *AuthService) revokeSessions(ctx context.Context, kullaniciID bson.ObjectID) {
	if err := s.sessions.RevokeAllForUser(ctx, kullaniciID, repository.RevokePasswordReset); err != nil {
		log.Printf("oturumlar iptal edilemedi: %v", err)
	}
}

func (s *AuthService) issueVerification(ctx context.Context, user *repository.User) error {
	if err := s.tokens.InvalidateUnused(ctx, user.ID, repository.PurposeEmailVerification); err != nil {
		return err
	}
	plain, hash, err := auth.NewToken()
	if err != nil {
		return err
	}
	doc := repository.VerificationToken{
		UserID:    user.ID,
		Token:     hash,
		Purpose:   repository.PurposeEmailVerification,
		ExpiresAt: time.Now().UTC().Add(auth.TokenTTL),
		Used:      false,
	}
	if err := s.tokens.Create(ctx, &doc); err != nil {
		return err
	}
	if err := s.mailer.Send(user.Email, verificationSubject, dogrulamaGovde(plain)); err != nil {
		log.Printf("doğrulama iletisi gönderilemedi: %v", err)
	}
	return nil
}

func dogrulamaGovde(token string) string {
	return fmt.Sprintf("Kontrata e-posta doğrulama\n\nDoğrulama kodunuz:\n\n%s\n\nBu kod 24 saat geçerlidir.\n", token)
}

func (s *AuthService) writeAudit(ctx context.Context, userID *bson.ObjectID, event, detail string) {
	meta := auth.MetaFrom(ctx)
	rec := repository.AuditRecord{
		Event:      event,
		IPAddress:  meta.IP,
		UserAgent:  meta.UserAgent,
		OccurredAt: time.Now().UTC(),
		Detail:     detail,
	}
	if userID != nil {
		rec.UserID = *userID
	}
	if err := s.audit.Insert(ctx, &rec); err != nil {
		log.Printf("denetim kaydı yazılamadı: %v", err)
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
	key := emailKey(epostaNorm)
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if prev, ok := l.last[key]; ok && now.Sub(prev) < l.every {
		return false
	}
	l.last[key] = now
	return true
}

func emailKey(eposta string) string {
	sum := sha256.Sum256([]byte(eposta))
	return hex.EncodeToString(sum[:])
}
