package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const yeniCihazKonu = "Yeni cihazdan giriş"

func (s *AuthService) rememberDevice(ctx context.Context, kullaniciID bson.ObjectID, user *repository.Kullanici) (bson.ObjectID, error) {
	meta := auth.MetaFrom(ctx)
	hash := auth.DeviceFingerprint(meta.DeviceID, meta.UserAgent, meta.AcceptLanguage)
	now := time.Now().UTC()
	existing, err := s.devices.GetByUserAndFingerprint(ctx, kullaniciID, hash)
	if err == nil {
		if err := s.devices.Touch(ctx, existing.ID, now, meta.IP, meta.UserAgent); err != nil {
			return bson.ObjectID{}, err
		}
		return existing.ID, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return bson.ObjectID{}, err
	}
	doc := repository.Cihaz{
		KullaniciID:    kullaniciID,
		CihazParmakIzi: hash,
		Ad:             auth.DeviceLabel(meta.UserAgent),
		Guvenilir:      false,
		IlkGorulme:     now,
		SonGorulme:     now,
		IPAdresi:       meta.IP,
		KullaniciAjani: meta.UserAgent,
	}
	if err := s.devices.Create(ctx, &doc); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			again, gerr := s.devices.GetByUserAndFingerprint(ctx, kullaniciID, hash)
			if gerr != nil {
				return bson.ObjectID{}, gerr
			}
			if err := s.devices.Touch(ctx, again.ID, now, meta.IP, meta.UserAgent); err != nil {
				return bson.ObjectID{}, err
			}
			return again.ID, nil
		}
		return bson.ObjectID{}, err
	}
	if user != nil {
		govde := "Kontrata güvenlik bildirimi\n\nhesabınıza yeni bir cihazdan giriş yapıldı\n\nCihaz: " + doc.Ad + "\n"
		if err := s.mailer.Gonder(user.Eposta, yeniCihazKonu, govde); err != nil {
			log.Printf("yeni cihaz iletisi gönderilemedi: %v", err)
		}
	}
	return doc.ID, nil
}

// Cihazlarim kullanıcının kayıtlı cihazlarını listeler.
func (s *AuthService) Cihazlarim(ctx context.Context) ([]*model.Cihaz, error) {
	id, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, auth.ErrUnauthorized
	}
	docs, err := s.devices.ListByUser(ctx, id.UserID)
	if err != nil {
		log.Printf("cihaz listesi alınamadı: %v", err)
		return nil, err
	}
	out := make([]*model.Cihaz, 0, len(docs))
	for i := range docs {
		out = append(out, toCihazModel(&docs[i]))
	}
	return out, nil
}

// CihazAdlandir cihazın görünen adını değiştirir.
func (s *AuthService) CihazAdlandir(ctx context.Context, id, ad string) (*model.Cihaz, error) {
	doc, err := s.ownedCihaz(ctx, id)
	if err != nil {
		return nil, err
	}
	ad = strings.TrimSpace(ad)
	if ad == "" {
		return nil, auth.ErrInvalidName
	}
	if err := s.devices.Rename(ctx, doc.ID, ad); err != nil {
		log.Printf("cihaz adı güncellenemedi: %v", err)
		return nil, err
	}
	doc.Ad = ad
	return toCihazModel(doc), nil
}

// CihazKaldir cihazı siler ve oturumlarını iptal eder.
func (s *AuthService) CihazKaldir(ctx context.Context, id string) (bool, error) {
	doc, err := s.ownedCihaz(ctx, id)
	if err != nil {
		return false, err
	}
	if err := s.sessions.RevokeByCihaz(ctx, doc.ID, repository.IptalCihazKaldir); err != nil {
		log.Printf("cihaz oturumları iptal edilemedi: %v", err)
		return false, err
	}
	if err := s.devices.Delete(ctx, doc.ID); err != nil {
		log.Printf("cihaz silinemedi: %v", err)
		return false, err
	}
	return true, nil
}

// CihazGuvenilirYap cihazı güvenilir işaretler.
func (s *AuthService) CihazGuvenilirYap(ctx context.Context, id string) (*model.Cihaz, error) {
	doc, err := s.ownedCihaz(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.devices.SetTrusted(ctx, doc.ID); err != nil {
		log.Printf("cihaz güvenilir yapılamadı: %v", err)
		return nil, err
	}
	doc.Guvenilir = true
	return toCihazModel(doc), nil
}

func (s *AuthService) ownedCihaz(ctx context.Context, id string) (*repository.Cihaz, error) {
	ident, ok := auth.IdentityFrom(ctx)
	if !ok {
		return nil, auth.ErrUnauthorized
	}
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, repository.ErrNotFound
	}
	doc, err := s.devices.GetByID(ctx, oid)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	if doc.KullaniciID != ident.UserID {
		return nil, repository.ErrNotFound
	}
	return doc, nil
}

func toCihazModel(d *repository.Cihaz) *model.Cihaz {
	out := &model.Cihaz{
		ID:         d.ID.Hex(),
		Ad:         d.Ad,
		Guvenilir:  d.Guvenilir,
		IlkGorulme: d.IlkGorulme,
		SonGorulme: d.SonGorulme,
	}
	if d.IPAdresi != "" {
		ip := d.IPAdresi
		out.IPAdresi = &ip
	}
	if d.KullaniciAjani != "" {
		ua := d.KullaniciAjani
		out.KullaniciAjani = &ua
	}
	return out
}
