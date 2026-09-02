package service

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const (
	mfaKonu        = "Giriş doğrulama kodu"
	loginWindow    = 15 * time.Minute
	loginMaxFails  = 10
	girisAyniYanit = true
)

// GirisYap şifre doğruysa MFA kodu gönderir; oturum açmaz.
// Yanlış şifre ve olmayan hesap aynı yanıtı verir.
func (s *AuthService) GirisYap(ctx context.Context, eposta, sifre string) (*model.GirisSonucu, error) {
	now := time.Now().UTC()
	dummy, err := s.dummyPending(now)
	if err != nil {
		return nil, err
	}
	sonuc := &model.GirisSonucu{MfaGerekli: girisAyniYanit, GeciciToken: dummy}

	norm, nerr := auth.NormalizeEposta(eposta)
	key := ""
	if nerr == nil {
		key = epostaKey(norm)
	}
	if key != "" && s.loginGuard.locked(key, now) {
		s.writeAudit(ctx, nil, repository.OlayGirisBasarisiz, "kilitli")
		return sonuc, nil
	}

	var user *repository.Kullanici
	if nerr == nil {
		u, gerr := s.users.GetByEposta(ctx, norm)
		if gerr != nil && !errors.Is(gerr, repository.ErrNotFound) {
			log.Printf("giriş okuma başarısız")
			return nil, gerr
		}
		user = u
	}
	hash := s.dummyHash
	var uid *bson.ObjectID
	if user != nil {
		hash = user.SifreHash
		uid = &user.ID
	}
	ver := auth.VerifyPassword(sifre, hash)
	ok := user != nil && user.EpostaDogrulandi && user.Durum != repository.DurumAskida && ver == nil
	if !ok {
		if key != "" {
			if s.loginGuard.fail(key, now) {
				s.writeAudit(ctx, uid, repository.OlayHesapKilitlendi, "")
			}
		}
		s.writeAudit(ctx, uid, repository.OlayGirisBasarisiz, "")
		return sonuc, nil
	}

	s.loginGuard.clear(key)
	if err := s.issueMFA(ctx, user); err != nil {
		log.Printf("MFA kodu hazırlanamadı")
		return sonuc, nil
	}
	pending, err := s.jwt.SignPending(user.ID.Hex(), now)
	if err != nil {
		log.Printf("geçici jeton üretilemedi")
		return sonuc, nil
	}
	s.writeAudit(ctx, &user.ID, repository.OlayGirisBasarili, "")
	return &model.GirisSonucu{MfaGerekli: true, GeciciToken: pending}, nil
}

// MFADogrula kodu doğrular ve oturum açar.
func (s *AuthService) MFADogrula(ctx context.Context, geciciToken, kod string) (*model.OturumSonucu, error) {
	uidHex, err := s.jwt.ParsePending(geciciToken)
	if err != nil {
		s.writeAudit(ctx, nil, repository.OlayMFABasarisiz, "")
		return nil, auth.ErrMFAFailed
	}
	oid, err := bson.ObjectIDFromHex(uidHex)
	if err != nil {
		s.writeAudit(ctx, nil, repository.OlayMFABasarisiz, "")
		return nil, auth.ErrMFAFailed
	}
	now := time.Now().UTC()
	doc, err := s.mfa.GetActive(ctx, oid, now)
	if err != nil {
		s.writeAudit(ctx, &oid, repository.OlayMFABasarisiz, "")
		return nil, auth.ErrMFAFailed
	}
	if !auth.MFACodeMatch(kod, doc.KodHash) {
		if err := s.mfa.RegisterFailure(ctx, doc.ID); err != nil {
			log.Printf("MFA deneme güncellenemedi")
		}
		s.writeAudit(ctx, &oid, repository.OlayMFABasarisiz, "")
		return nil, auth.ErrMFAFailed
	}
	if err := s.mfa.MarkUsed(ctx, doc.ID); err != nil {
		log.Printf("MFA kodu kapatılamadı")
		return nil, auth.ErrMFAFailed
	}
	out, err := s.openSession(ctx, oid)
	if err != nil {
		return nil, err
	}
	s.writeAudit(ctx, &oid, repository.OlayMFABasarili, "")
	return out, nil
}

// JetonYenile eski yenileme jetonunu iptal edip yenisini verir.
func (s *AuthService) JetonYenile(ctx context.Context, yenileme string) (*model.OturumSonucu, error) {
	if yenileme == "" {
		return nil, auth.ErrGecersizYenilemeJetonu
	}
	now := time.Now().UTC()
	old, err := s.sessions.GetByRefreshHash(ctx, auth.HashToken(yenileme), now)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, auth.ErrGecersizYenilemeJetonu
		}
		log.Printf("yenileme oturumu okunamadı")
		return nil, err
	}
	if err := s.sessions.Revoke(ctx, old.ID, repository.IptalYenileme); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, auth.ErrGecersizYenilemeJetonu
		}
		log.Printf("eski oturum iptal edilemedi")
		return nil, err
	}
	out, err := s.openSession(ctx, old.KullaniciID)
	if err != nil {
		return nil, err
	}
	s.writeAudit(ctx, &old.KullaniciID, repository.OlayOturumYenilendi, "")
	return out, nil
}

// CikisYap mevcut oturumu iptal eder.
func (s *AuthService) CikisYap(ctx context.Context) (bool, error) {
	id, ok := auth.IdentityFrom(ctx)
	if !ok {
		return false, auth.ErrUnauthorized
	}
	if err := s.sessions.Revoke(ctx, id.SessionID, repository.IptalCikis); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, auth.ErrUnauthorized
		}
		log.Printf("çıkış başarısız")
		return false, err
	}
	s.writeAudit(ctx, &id.UserID, repository.OlayCikis, "")
	return true, nil
}

// Oturumlarim kullanıcının aktif oturumlarını listeler.
func (s *AuthService) Oturumlarim(ctx context.Context) ([]*model.OturumBilgisi, error) {
	id, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, auth.ErrUnauthorized
	}
	now := time.Now().UTC()
	docs, err := s.sessions.ListActiveByUser(ctx, id.UserID, now)
	if err != nil {
		log.Printf("oturum listesi alınamadı")
		return nil, err
	}
	out := make([]*model.OturumBilgisi, 0, len(docs))
	for i := range docs {
		d := docs[i]
		ip, ua := d.IPAdresi, d.KullaniciAjani
		item := &model.OturumBilgisi{
			ID:              d.ID.Hex(),
			OlusturmaTarihi: d.OlusturmaTarihi,
			SonKullanma:     d.SonKullanma,
			MevcutMu:        d.ID == id.SessionID,
		}
		if ip != "" {
			item.IPAdresi = &ip
		}
		if ua != "" {
			item.KullaniciAjani = &ua
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *AuthService) openSession(ctx context.Context, kullaniciID bson.ObjectID) (*model.OturumSonucu, error) {
	plain, hash, err := auth.NewToken()
	if err != nil {
		log.Printf("yenileme jetonu üretilemedi")
		return nil, repository.ErrStore
	}
	now := time.Now().UTC()
	meta := auth.MetaFrom(ctx)
	doc := repository.Oturum{
		KullaniciID:       kullaniciID,
		YenilemeTokenHash: hash,
		OlusturmaTarihi:   now,
		SonKullanma:       now.Add(auth.RefreshTTL),
		IptalEdildi:       false,
		IPAdresi:          meta.IP,
		KullaniciAjani:    meta.UserAgent,
	}
	if err := s.sessions.Create(ctx, &doc); err != nil {
		log.Printf("oturum yazılamadı")
		return nil, err
	}
	access, err := s.jwt.SignAccess(kullaniciID.Hex(), doc.ID.Hex(), now)
	if err != nil {
		log.Printf("erişim jetonu üretilemedi")
		return nil, repository.ErrStore
	}
	return &model.OturumSonucu{ErisimJetonu: access, YenilemeJetonu: plain}, nil
}

func (s *AuthService) issueMFA(ctx context.Context, user *repository.Kullanici) error {
	if err := s.mfa.InvalidateUnused(ctx, user.ID); err != nil {
		return err
	}
	plain, hash, err := auth.NewMFACode()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	doc := repository.MFAKodu{
		KullaniciID:     user.ID,
		KodHash:         hash,
		SonKullanma:     now.Add(auth.MFATTL),
		Kullanildi:      false,
		DenemeSayisi:    0,
		OlusturmaTarihi: now,
	}
	if err := s.mfa.Create(ctx, &doc); err != nil {
		return err
	}
	govde := "Kontrata giriş doğrulama\n\nGiriş kodunuz:\n\n" + plain + "\n\nBu kod 2 dakika geçerlidir.\n"
	if err := s.mailer.Gonder(user.Eposta, mfaKonu, govde); err != nil {
		log.Printf("MFA iletisi gönderilemedi")
	}
	return nil
}

func (s *AuthService) dummyPending(now time.Time) (string, error) {
	tok, err := s.jwt.SignPending(bson.NewObjectID().Hex(), now)
	if err != nil {
		log.Printf("geçici jeton üretilemedi")
		return "", repository.ErrStore
	}
	return tok, nil
}

// BearerMiddleware Authorization Bearer jetonunu çözüp kimliği bağlama yazar.
func (s *AuthService) BearerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		typ, tok, ok := strings.Cut(strings.TrimSpace(r.Header.Get("Authorization")), " ")
		if !ok || !strings.EqualFold(typ, "Bearer") || tok == "" {
			next.ServeHTTP(w, r)
			return
		}
		uidHex, sidHex, err := s.jwt.ParseAccess(tok)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		uid, err := bson.ObjectIDFromHex(uidHex)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		sid, err := bson.ObjectIDFromHex(sidHex)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		ses, err := s.sessions.GetByID(r.Context(), sid)
		if err != nil || ses.IptalEdildi || ses.KullaniciID != uid || !ses.SonKullanma.After(time.Now().UTC()) {
			next.ServeHTTP(w, r)
			return
		}
		ctx := auth.WithIdentity(r.Context(), auth.Identity{UserID: uid, SessionID: sid})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type loginGuard struct {
	mu    sync.Mutex
	fails map[string][]time.Time
}

func newLoginGuard() *loginGuard {
	return &loginGuard{fails: make(map[string][]time.Time)}
}

func (g *loginGuard) prune(stamps []time.Time, now time.Time) []time.Time {
	cut := now.Add(-loginWindow)
	out := stamps[:0]
	for _, t := range stamps {
		if t.After(cut) {
			out = append(out, t)
		}
	}
	return out
}

func (g *loginGuard) locked(key string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.fails[key] = g.prune(g.fails[key], now)
	return len(g.fails[key]) >= loginMaxFails
}

func (g *loginGuard) fail(key string, now time.Time) (justLocked bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.fails[key] = g.prune(append(g.fails[key], now), now)
	return len(g.fails[key]) == loginMaxFails
}

func (g *loginGuard) clear(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.fails, key)
}
