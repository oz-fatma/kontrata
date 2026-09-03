package service

import (
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/mailer"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const inviteSubject = "Organizasyon daveti"

func roleFromModel(r model.Rol) string {
	return string(r)
}

func toRole(s string) model.Rol {
	switch s {
	case repository.RoleAdmin:
		return model.RolYonetici
	case repository.RoleViewer:
		return model.RolGoruntuleyici
	default:
		return model.RolSahip
	}
}

func toAccountType(s string) model.HesapTipi {
	if s == repository.AccountCorporate {
		return model.HesapTipiKurumsal
	}
	return model.HesapTipiBireysel
}

func toOrgStatus(s string) model.OrganizasyonDurumu {
	if s == repository.OrgStatusSuspended {
		return model.OrganizasyonDurumuAskida
	}
	return model.OrganizasyonDurumuAktif
}

func toMember(u *repository.User) *model.Uye {
	return &model.Uye{
		ID:        u.ID.Hex(),
		Eposta:    u.Email,
		Rol:       toRole(u.Role),
		HesapTipi: toAccountType(u.AccountType),
	}
}

func toOrganization(o *repository.Organization) *model.Organizasyon {
	out := &model.Organizasyon{
		ID:              o.ID.Hex(),
		Ad:              o.Name,
		Durum:           toOrgStatus(o.Status),
		OlusturmaTarihi: o.CreatedAt,
	}
	if o.TaxID != "" {
		v := o.TaxID
		out.VergiNo = &v
	}
	return out
}

// Organizasyonum kullanıcının kurumunu döner; bireyselde null.
func (s *AuthService) MyOrganization(ctx context.Context) (*model.Organizasyon, error) {
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
	return toOrganization(org), nil
}

// Uyeler organizasyon üyelerini listeler.
func (s *AuthService) Members(ctx context.Context) ([]*model.Uye, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opMemberView) {
		return nil, auth.ErrForbidden
	}
	if !act.hasOrg() {
		return []*model.Uye{toMember(&act.user)}, nil
	}
	docs, err := s.users.ListByOrg(ctx, act.orgID())
	if err != nil {
		log.Printf("üye listesi alınamadı: %v", err)
		return nil, err
	}
	out := make([]*model.Uye, 0, len(docs))
	for i := range docs {
		out = append(out, toMember(&docs[i]))
	}
	return out, nil
}

// UyeDavetEt e-posta ile üyelik daveti gönderir.
func (s *AuthService) InviteMember(ctx context.Context, eposta string, rol model.Rol) (bool, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return false, err
	}
	if !act.can(opMemberManage) || !act.hasOrg() {
		return false, auth.ErrForbidden
	}
	r := roleFromModel(rol)
	if r == repository.RoleOwner || r == "" {
		return false, auth.ErrForbidden
	}
	norm, nerr := auth.NormalizeEmail(eposta)
	if nerr != nil {
		return true, nil
	}
	if existing, gerr := s.users.GetByEmail(ctx, norm); gerr == nil && existing.OrganizationID == act.orgID() {
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
	doc := repository.Invite{
		OrganizationID: act.orgID(),
		Email:          norm,
		Role:           r,
		TokenHash:      hash,
		ExpiresAt:      time.Now().UTC().Add(auth.InviteTTL),
		Used:           false,
		InvitedBy:      act.user.ID,
	}
	if err := s.davets.Create(ctx, &doc); err != nil {
		log.Printf("davet yazılamadı: %v", err)
		return false, err
	}
	if err := s.mailer.Send(norm, inviteSubject, mailer.InviteBody(plain)); err != nil {
		log.Printf("davet iletisi gönderilemedi: %v", err)
	}
	s.writeAudit(ctx, &act.user.ID, repository.EventMemberInvited, r)
	return true, nil
}

// DavetiKabulEt davet koduyla yeni üyeyi oluşturur.
func (s *AuthService) AcceptInvite(ctx context.Context, token, sifre string) (bool, error) {
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
	if existing, gerr := s.users.GetByEmail(ctx, davet.Email); gerr == nil {
		if existing.OrganizationID == davet.OrganizationID {
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
	user := repository.User{
		Email:          consumed.Email,
		PasswordHash:   hash,
		EmailVerified:  true,
		Status:         repository.StatusActive,
		AccountType:    repository.AccountCorporate,
		OrganizationID: consumed.OrganizationID,
		Role:           consumed.Role,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.users.Create(ctx, &user); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return true, nil
		}
		log.Printf("davet kaydı başarısız: %v", err)
		return false, err
	}
	s.writeAudit(ctx, &user.ID, repository.EventRegister, "davet")
	return true, nil
}

// UyeRolDegistir üyenin rolünü değiştirir. SAHIP ataması sahipliği devreder.
func (s *AuthService) ChangeMemberRole(ctx context.Context, kullaniciID string, rol model.Rol) (*model.Uye, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opMemberManage) || !act.hasOrg() {
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
	if hedef.OrganizationID != act.orgID() {
		return nil, repository.ErrNotFound
	}
	yeni := roleFromModel(rol)
	if yeni == "" {
		return nil, auth.ErrForbidden
	}
	if yeni == repository.RoleOwner {
		if hedef.ID == act.user.ID {
			return toMember(hedef), nil
		}
		if err := s.users.SetRole(ctx, hedef.ID, repository.RoleOwner); err != nil {
			return nil, err
		}
		if err := s.users.SetRole(ctx, act.user.ID, repository.RoleAdmin); err != nil {
			return nil, err
		}
		if err := s.orgs.SetOwner(ctx, act.orgID(), hedef.ID); err != nil {
			return nil, err
		}
		hedef.Role = repository.RoleOwner
		s.writeAudit(ctx, &act.user.ID, repository.EventRoleChanged, "devir")
		return toMember(hedef), nil
	}
	if hedef.Role == repository.RoleOwner {
		return nil, auth.ErrTransferOwnership
	}
	if err := s.users.SetRole(ctx, hedef.ID, yeni); err != nil {
		log.Printf("rol güncellenemedi: %v", err)
		return nil, err
	}
	hedef.Role = yeni
	s.writeAudit(ctx, &act.user.ID, repository.EventRoleChanged, yeni)
	return toMember(hedef), nil
}

// UyeCikar üyeyi organizasyondan ayırır.
func (s *AuthService) RemoveMember(ctx context.Context, kullaniciID string) (bool, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return false, err
	}
	if !act.can(opMemberManage) || !act.hasOrg() {
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
	if hedef.OrganizationID != act.orgID() {
		return false, repository.ErrNotFound
	}
	if hedef.Role == repository.RoleOwner {
		return false, auth.ErrTransferOwnership
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
		if err := s.deleteOrganization(ctx, act.orgID()); err != nil {
			return false, err
		}
	}
	s.writeAudit(ctx, &act.user.ID, repository.EventMemberRemoved, "")
	return true, nil
}

// OrganizasyonSil kurumu, sözleşmelerini ve davetleri kaldırır; üye bağını keser.
func (s *AuthService) DeleteOrganization(ctx context.Context) (bool, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return false, err
	}
	if !act.can(opOrgDelete) || !act.hasOrg() {
		return false, auth.ErrForbidden
	}
	if err := s.deleteOrganization(ctx, act.orgID()); err != nil {
		log.Printf("organizasyon silinemedi: %v", err)
		return false, err
	}
	s.writeAudit(ctx, &act.user.ID, repository.EventOrganizationDeleted, "")
	return true, nil
}

func (s *AuthService) deleteOrganization(ctx context.Context, orgID bson.ObjectID) error {
	return s.deleteOrg(ctx, orgID, true)
}

func (s *AuthService) deleteOrg(ctx context.Context, orgID bson.ObjectID, removeFiles bool) error {
	var ids []string
	if removeFiles {
		var err error
		ids, err = s.storedFilesByOrg(ctx, orgID)
		if err != nil {
			return err
		}
	}
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
	if err := s.deleteOrgLLM(ctx, orgID); err != nil {
		return err
	}
	if removeFiles {
		s.removeStoredFiles(ids)
	}
	return nil
}

func (s *AuthService) storedFilesByUser(ctx context.Context, userID bson.ObjectID) ([]string, error) {
	docs, err := s.soz.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return storedFileIDs(docs), nil
}

func (s *AuthService) storedFilesByOrg(ctx context.Context, orgID bson.ObjectID) ([]string, error) {
	docs, err := s.soz.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return storedFileIDs(docs), nil
}

func (s *AuthService) removeStoredFiles(ids []string) {
	if s.files == nil || len(ids) == 0 {
		return
	}
	if err := s.files.RemoveAll(ids); err != nil {
		log.Printf("sozlesme dosyaları silinemedi: %v", err)
	}
}

func (s *AuthService) ownerHasOtherMembers(ctx context.Context, user *repository.User) (bool, error) {
	if user.Role != repository.RoleOwner || user.OrganizationID.IsZero() {
		return false, nil
	}
	n, err := s.users.CountByOrg(ctx, user.OrganizationID)
	if err != nil {
		return false, err
	}
	return n > 1, nil
}
