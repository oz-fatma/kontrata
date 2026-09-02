package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const davetKonu = "Organizasyon daveti"

func rolFromModel(r model.Rol) string {
	return string(r)
}

func toRol(s string) model.Rol {
	switch s {
	case repository.RolYonetici:
		return model.RolYonetici
	case repository.RolGoruntuleyici:
		return model.RolGoruntuleyici
	default:
		return model.RolSahip
	}
}

func toHesapTipi(s string) model.HesapTipi {
	if s == repository.HesapKurumsal {
		return model.HesapTipiKurumsal
	}
	return model.HesapTipiBireysel
}

func toOrgDurum(s string) model.OrganizasyonDurumu {
	if s == repository.OrgDurumAskida {
		return model.OrganizasyonDurumuAskida
	}
	return model.OrganizasyonDurumuAktif
}

func toUye(u *repository.Kullanici) *model.Uye {
	return &model.Uye{
		ID:        u.ID.Hex(),
		Eposta:    u.Eposta,
		Rol:       toRol(u.Rol),
		HesapTipi: toHesapTipi(u.HesapTipi),
	}
}

func toOrganizasyon(o *repository.Organizasyon) *model.Organizasyon {
	out := &model.Organizasyon{
		ID:              o.ID.Hex(),
		Ad:              o.Ad,
		Durum:           toOrgDurum(o.Durum),
		OlusturmaTarihi: o.OlusturmaTarihi,
	}
	if o.VergiNo != "" {
		v := o.VergiNo
		out.VergiNo = &v
	}
	return out
}

// Organizasyonum kullanıcının kurumunu döner; bireyselde null.
func (s *AuthService) Organizasyonum(ctx context.Context) (*model.Organizasyon, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.hasOrg() {
		return nil, nil
	}
	org, err := s.orgs.GetByID(ctx, act.orgID())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		log.Printf("organizasyon okunamadı: %v", err)
		return nil, err
	}
	return toOrganizasyon(org), nil
}

// Uyeler organizasyon üyelerini listeler.
func (s *AuthService) Uyeler(ctx context.Context) ([]*model.Uye, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opUyeGor) {
		return nil, auth.ErrForbidden
	}
	if !act.hasOrg() {
		return []*model.Uye{toUye(&act.user)}, nil
	}
	docs, err := s.users.ListByOrg(ctx, act.orgID())
	if err != nil {
		log.Printf("üye listesi alınamadı: %v", err)
		return nil, err
	}
	out := make([]*model.Uye, 0, len(docs))
	for i := range docs {
		out = append(out, toUye(&docs[i]))
	}
	return out, nil
}

// UyeDavetEt e-posta ile üyelik daveti gönderir.
func (s *AuthService) UyeDavetEt(ctx context.Context, eposta string, rol model.Rol) (bool, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return false, err
	}
	if !act.can(opUyeYonet) || !act.hasOrg() {
		return false, auth.ErrForbidden
	}
	r := rolFromModel(rol)
	if r == repository.RolSahip || r == "" {
		return false, auth.ErrForbidden
	}
	norm, nerr := auth.NormalizeEposta(eposta)
	if nerr != nil {
		return true, nil
	}
	if existing, gerr := s.users.GetByEposta(ctx, norm); gerr == nil && existing.OrganizasyonID == act.orgID() {
		return true, nil
	} else if gerr != nil && !errors.Is(gerr, repository.ErrNotFound) {
		log.Printf("üye daveti okunamadı: %v", gerr)
		return false, gerr
	}
	if err := s.davets.InvalidateUnused(ctx, act.orgID(), norm); err != nil {
		log.Printf("eski davet kapatılamadı: %v", err)
		return false, err
	}
	plain, hash, err := auth.NewToken()
	if err != nil {
		return false, err
	}
	doc := repository.Davet{
		OrganizasyonID:     act.orgID(),
		Eposta:             norm,
		Rol:                r,
		TokenHash:          hash,
		SonKullanma:        time.Now().UTC().Add(auth.InviteTTL),
		Kullanildi:         false,
		DavetEdenKullanici: act.user.ID,
	}
	if err := s.davets.Create(ctx, &doc); err != nil {
		log.Printf("davet yazılamadı: %v", err)
		return false, err
	}
	govde := fmt.Sprintf("Kontrata organizasyon daveti\n\nDavet kodunuz:\n\n%s\n\nBu kod 7 gün geçerlidir.\n", plain)
	if err := s.mailer.Gonder(norm, davetKonu, govde); err != nil {
		log.Printf("davet iletisi gönderilemedi: %v", err)
	}
	s.writeAudit(ctx, &act.user.ID, repository.OlayUyeDavet, r)
	return true, nil
}

// DavetiKabulEt davet koduyla yeni üyeyi oluşturur.
func (s *AuthService) DavetiKabulEt(ctx context.Context, token, sifre string) (bool, error) {
	if token == "" {
		return false, nil
	}
	hash, err := auth.HashPassword(sifre, s.params)
	if err != nil {
		return false, err
	}
	davet, err := s.davets.GetByHash(ctx, auth.HashToken(token), time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		log.Printf("davet okunamadı: %v", err)
		return false, err
	}
	if existing, gerr := s.users.GetByEposta(ctx, davet.Eposta); gerr == nil {
		if existing.OrganizasyonID == davet.OrganizasyonID {
			_, _ = s.davets.Consume(ctx, auth.HashToken(token), time.Now().UTC())
			return true, nil
		}
		return true, nil
	} else if gerr != nil && !errors.Is(gerr, repository.ErrNotFound) {
		return false, gerr
	}
	consumed, err := s.davets.Consume(ctx, auth.HashToken(token), time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		log.Printf("davet kapatılamadı: %v", err)
		return false, err
	}
	now := time.Now().UTC()
	user := repository.Kullanici{
		Eposta:           consumed.Eposta,
		SifreHash:        hash,
		EpostaDogrulandi: true,
		Durum:            repository.DurumAktif,
		HesapTipi:        repository.HesapKurumsal,
		OrganizasyonID:   consumed.OrganizasyonID,
		Rol:              consumed.Rol,
		OlusturmaTarihi:  now,
		GuncellemeTarihi: now,
	}
	if err := s.users.Create(ctx, &user); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return true, nil
		}
		log.Printf("davet kaydı başarısız: %v", err)
		return false, err
	}
	s.writeAudit(ctx, &user.ID, repository.OlayKayit, "davet")
	return true, nil
}

// UyeRolDegistir üyenin rolünü değiştirir. SAHIP ataması sahipliği devreder.
func (s *AuthService) UyeRolDegistir(ctx context.Context, kullaniciID string, rol model.Rol) (*model.Uye, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opUyeYonet) || !act.hasOrg() {
		return nil, auth.ErrForbidden
	}
	oid, err := bson.ObjectIDFromHex(kullaniciID)
	if err != nil {
		return nil, repository.ErrNotFound
	}
	hedef, err := s.users.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	if hedef.OrganizasyonID != act.orgID() {
		return nil, repository.ErrNotFound
	}
	yeni := rolFromModel(rol)
	if yeni == "" {
		return nil, auth.ErrForbidden
	}
	if yeni == repository.RolSahip {
		if hedef.ID == act.user.ID {
			return toUye(hedef), nil
		}
		if err := s.users.SetRol(ctx, hedef.ID, repository.RolSahip); err != nil {
			return nil, err
		}
		if err := s.users.SetRol(ctx, act.user.ID, repository.RolYonetici); err != nil {
			return nil, err
		}
		if err := s.orgs.SetSahip(ctx, act.orgID(), hedef.ID); err != nil {
			return nil, err
		}
		hedef.Rol = repository.RolSahip
		s.writeAudit(ctx, &act.user.ID, repository.OlayRolDegistir, "devir")
		return toUye(hedef), nil
	}
	if hedef.Rol == repository.RolSahip {
		return nil, auth.ErrSahipDevret
	}
	if err := s.users.SetRol(ctx, hedef.ID, yeni); err != nil {
		log.Printf("rol güncellenemedi: %v", err)
		return nil, err
	}
	hedef.Rol = yeni
	s.writeAudit(ctx, &act.user.ID, repository.OlayRolDegistir, yeni)
	return toUye(hedef), nil
}

// UyeCikar üyeyi organizasyondan ayırır.
func (s *AuthService) UyeCikar(ctx context.Context, kullaniciID string) (bool, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return false, err
	}
	if !act.can(opUyeYonet) || !act.hasOrg() {
		return false, auth.ErrForbidden
	}
	oid, err := bson.ObjectIDFromHex(kullaniciID)
	if err != nil {
		return false, repository.ErrNotFound
	}
	hedef, err := s.users.GetByID(ctx, oid)
	if err != nil {
		return false, err
	}
	if hedef.OrganizasyonID != act.orgID() {
		return false, repository.ErrNotFound
	}
	if hedef.Rol == repository.RolSahip {
		return false, auth.ErrSahipDevret
	}
	if err := s.users.DetachOrg(ctx, hedef.ID); err != nil {
		log.Printf("üye çıkarılamadı: %v", err)
		return false, err
	}
	n, err := s.users.CountByOrg(ctx, act.orgID())
	if err != nil {
		return false, err
	}
	if n == 0 {
		if err := s.silOrganizasyon(ctx, act.orgID()); err != nil {
			return false, err
		}
	}
	s.writeAudit(ctx, &act.user.ID, repository.OlayUyeCikar, "")
	return true, nil
}

// OrganizasyonSil kurumu, sözleşmelerini ve davetleri kaldırır; üye bağını keser.
func (s *AuthService) OrganizasyonSil(ctx context.Context) (bool, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return false, err
	}
	if !act.can(opOrgSil) || !act.hasOrg() {
		return false, auth.ErrForbidden
	}
	if err := s.silOrganizasyon(ctx, act.orgID()); err != nil {
		log.Printf("organizasyon silinemedi: %v", err)
		return false, err
	}
	s.writeAudit(ctx, &act.user.ID, repository.OlayOrganizasyonSilindi, "")
	return true, nil
}

func (s *AuthService) silOrganizasyon(ctx context.Context, orgID bson.ObjectID) error {
	if err := s.soz.DeleteByOrg(ctx, orgID); err != nil {
		return err
	}
	if err := s.davets.DeleteByOrg(ctx, orgID); err != nil {
		return err
	}
	if err := s.users.DetachOrgByOrg(ctx, orgID); err != nil {
		return err
	}
	if err := s.orgs.Delete(ctx, orgID); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	return nil
}

func (s *AuthService) sahipBaskaUyeVar(ctx context.Context, user *repository.Kullanici) (bool, error) {
	if user.Rol != repository.RolSahip || user.OrganizasyonID.IsZero() {
		return false, nil
	}
	n, err := s.users.CountByOrg(ctx, user.OrganizasyonID)
	if err != nil {
		return false, err
	}
	return n > 1, nil
}
