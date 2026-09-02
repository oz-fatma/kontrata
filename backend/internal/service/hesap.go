package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const (
	hesapSilmeKonu = "Hesap silme onayı"
	deleteStepUser = "kullanici"
)

var errDeleteProbe = errors.New("silme adımı başarısız")

// HesapSilmeIste onay kodunu e-postaya gönderir.
func (s *AuthService) HesapSilmeIste(ctx context.Context) (bool, error) {
	id, ok := auth.IdentityFrom(ctx)
	if !ok {
		return false, auth.ErrUnauthorized
	}
	user, err := s.users.GetByID(ctx, id.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return true, nil
		}
		log.Printf("hesap silme isteği başarısız: %v", err)
		return false, err
	}
	if !s.silmeLimiter.allow(user.Eposta) {
		return true, nil
	}
	if err := s.issueAccountDelete(ctx, user); err != nil {
		log.Printf("hesap silme kodu hazırlanamadı: %v", err)
	}
	return true, nil
}

func (s *AuthService) issueAccountDelete(ctx context.Context, user *repository.Kullanici) error {
	if err := s.tokens.InvalidateUnused(ctx, user.ID, repository.AmacHesapSilme); err != nil {
		return err
	}
	plain, hash, err := auth.NewToken()
	if err != nil {
		return err
	}
	doc := repository.DogrulamaTokeni{
		KullaniciID: user.ID,
		Token:       hash,
		Amac:        repository.AmacHesapSilme,
		SonKullanma: time.Now().UTC().Add(auth.AccountDeleteTTL),
		Kullanildi:  false,
	}
	if err := s.tokens.Create(ctx, &doc); err != nil {
		return err
	}
	govde := fmt.Sprintf("Kontrata hesap silme\n\nHesap silme onay kodunuz:\n\n%s\n\nBu kod 1 saat geçerlidir.\n", plain)
	if err := s.mailer.Gonder(user.Eposta, hesapSilmeKonu, govde); err != nil {
		log.Printf("hesap silme iletisi gönderilemedi: %v", err)
	}
	return nil
}

// HesapSil onay kodu doğruysa kullanıcı verisini atomik siler.
func (s *AuthService) HesapSil(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	if s.db == nil {
		return false, repository.ErrUnavailable
	}
	var found bool
	err := s.db.WithTransaction(ctx, func(ctx context.Context) error {
		doc, err := s.tokens.Consume(ctx, auth.HashToken(token), repository.AmacHesapSilme, time.Now().UTC())
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil
			}
			return err
		}
		found = true
		return s.purgeAccount(ctx, doc.KullaniciID)
	})
	if err != nil {
		log.Printf("hesap silinemedi: %v", err)
		return false, err
	}
	return found, nil
}

func (s *AuthService) purgeAccount(ctx context.Context, kullaniciID bson.ObjectID) error {
	user, err := s.users.GetByID(ctx, kullaniciID)
	if err != nil {
		return err
	}
	if baska, err := s.sahipBaskaUyeVar(ctx, user); err != nil {
		return err
	} else if baska {
		return auth.ErrSahipDevret
	}
	if !user.OrganizasyonID.IsZero() {
		n, err := s.users.CountByOrg(ctx, user.OrganizasyonID)
		if err != nil {
			return err
		}
		if user.Rol == repository.RolSahip || n <= 1 {
			if err := s.silOrganizasyon(ctx, user.OrganizasyonID); err != nil {
				return err
			}
		} else if err := s.users.DetachOrg(ctx, kullaniciID); err != nil {
			return err
		}
	} else if err := s.soz.DeleteByUser(ctx, kullaniciID); err != nil {
		return err
	}
	if err := s.tokens.DeleteByUser(ctx, kullaniciID); err != nil {
		return err
	}
	if err := s.mfa.DeleteByUser(ctx, kullaniciID); err != nil {
		return err
	}
	if err := s.sessions.DeleteByUser(ctx, kullaniciID); err != nil {
		return err
	}
	if err := s.devices.DeleteByUser(ctx, kullaniciID); err != nil {
		return err
	}
	if err := s.audit.AnonymizeByUser(ctx, kullaniciID); err != nil {
		return err
	}
	if s.deleteFailAt == deleteStepUser {
		return errDeleteProbe
	}
	if err := s.users.Delete(ctx, kullaniciID); err != nil {
		return err
	}
	return s.audit.Insert(ctx, &repository.DenetimKaydi{
		KullaniciID:    repository.KullaniciSilinmis,
		Olay:           repository.OlayHesapSilindi,
		IPAdresi:       "",
		KullaniciAjani: "",
		Zaman:          time.Now().UTC(),
	})
}

// VerilerimiIndir kullanıcının dışa aktarılabilir verisini JSON döner. Hash ve token yok.
func (s *AuthService) VerilerimiIndir(ctx context.Context) (string, error) {
	id, ok := auth.IdentityFrom(ctx)
	if !ok {
		return "", auth.ErrUnauthorized
	}
	user, err := s.users.GetByID(ctx, id.UserID)
	if err != nil {
		return "", err
	}
	cihazlar, err := s.devices.ListByUser(ctx, id.UserID)
	if err != nil {
		return "", err
	}
	oturumlar, err := s.sessions.ListByUser(ctx, id.UserID)
	if err != nil {
		return "", err
	}
	sozlesmeler, err := s.soz.ListByUser(ctx, id.UserID)
	if err != nil {
		return "", err
	}
	if !user.OrganizasyonID.IsZero() {
		orgSoz, err := s.soz.ListByOrg(ctx, user.OrganizasyonID)
		if err != nil {
			return "", err
		}
		sozlesmeler = orgSoz
	}
	denetim, err := s.audit.ListByUser(ctx, id.UserID)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"eposta":           user.Eposta,
		"durum":            user.Durum,
		"epostaDogrulandi": user.EpostaDogrulandi,
		"olusturmaTarihi":  user.OlusturmaTarihi,
		"cihazlar":         exportCihazlar(cihazlar),
		"oturumlar":        exportOturumlar(oturumlar),
		"sozlesmeler":      exportSozlesmeler(sozlesmeler),
		"denetim":          exportDenetim(denetim),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		log.Printf("veri dışa aktarma başarısız: %v", err)
		return "", repository.ErrStore
	}
	return string(raw), nil
}

func exportCihazlar(docs []repository.Cihaz) []map[string]any {
	out := make([]map[string]any, 0, len(docs))
	for i := range docs {
		d := docs[i]
		out = append(out, map[string]any{
			"id":             d.ID.Hex(),
			"ad":             d.Ad,
			"guvenilir":      d.Guvenilir,
			"ilkGorulme":     d.IlkGorulme,
			"sonGorulme":     d.SonGorulme,
			"ipAdresi":       d.IPAdresi,
			"kullaniciAjani": d.KullaniciAjani,
		})
	}
	return out
}

func exportOturumlar(docs []repository.Oturum) []map[string]any {
	out := make([]map[string]any, 0, len(docs))
	for i := range docs {
		d := docs[i]
		item := map[string]any{
			"id":              d.ID.Hex(),
			"olusturmaTarihi": d.OlusturmaTarihi,
			"sonKullanma":     d.SonKullanma,
			"iptalEdildi":     d.IptalEdildi,
			"ipAdresi":        d.IPAdresi,
			"kullaniciAjani":  d.KullaniciAjani,
		}
		if !d.CihazID.IsZero() {
			item["cihazId"] = d.CihazID.Hex()
		}
		out = append(out, item)
	}
	return out
}

func exportSozlesmeler(docs []repository.Sozlesme) []map[string]any {
	out := make([]map[string]any, 0, len(docs))
	for i := range docs {
		d := docs[i]
		item := map[string]any{
			"id":               d.ID.Hex(),
			"olusturmaTarihi":  d.OlusturmaTarihi,
			"guncellemeTarihi": d.GuncellemeTarihi,
			"durum":            d.Durum,
		}
		if d.DosyaAdi != nil {
			item["dosyaAdi"] = *d.DosyaAdi
		}
		out = append(out, item)
	}
	return out
}

func exportDenetim(docs []repository.DenetimKaydi) []map[string]any {
	out := make([]map[string]any, 0, len(docs))
	for i := range docs {
		d := docs[i]
		out = append(out, map[string]any{
			"olay":  d.Olay,
			"zaman": d.Zaman,
			"detay": d.Detay,
		})
	}
	return out
}
