package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/mailer"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const (
	accountDeleteSubject = "Hesap silme onayı"
	deleteStepUser       = "kullanici"
)

var errDeleteProbe = errors.New("silme adımı başarısız")

// HesapSilmeIste onay kodunu e-postaya gönderir.
func (s *AuthService) RequestAccountDelete(ctx context.Context) (bool, error) {
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
	if !s.deleteLimiter.allow(user.Email) {
		return true, nil
	}
	if err := s.issueAccountDelete(ctx, user); err != nil {
		log.Printf("hesap silme kodu hazırlanamadı: %v", err)
	}
	return true, nil
}

func (s *AuthService) issueAccountDelete(ctx context.Context, user *repository.User) error {
	if err := s.tokens.InvalidateUnused(ctx, user.ID, repository.PurposeAccountDelete); err != nil {
		return err
	}
	plain, hash, err := auth.NewToken()
	if err != nil {
		return err
	}
	doc := repository.VerificationToken{
		UserID:    user.ID,
		Token:     hash,
		Purpose:   repository.PurposeAccountDelete,
		ExpiresAt: time.Now().UTC().Add(auth.AccountDeleteTTL),
		Used:      false,
	}
	if err := s.tokens.Create(ctx, &doc); err != nil {
		return err
	}
	if err := s.mailer.Send(user.Email, accountDeleteSubject, mailer.AccountDeleteBody(plain)); err != nil {
		log.Printf("hesap silme iletisi gönderilemedi: %v", err)
	}
	return nil
}

// HesapSil onay kodu doğruysa kullanıcı verisini atomik siler.
func (s *AuthService) DeleteAccount(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	if s.db == nil {
		return false, repository.ErrUnavailable
	}
	var found bool
	var fileIDs []string
	err := s.db.WithTransaction(ctx, func(ctx context.Context) error {
		doc, err := s.tokens.Consume(ctx, auth.HashToken(token), repository.PurposeAccountDelete, time.Now().UTC())
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil
			}
			return err
		}
		found = true
		ids, err := s.purgeAccount(ctx, doc.UserID)
		fileIDs = ids
		return err
	})
	if err != nil {
		log.Printf("hesap silinemedi: %v", err)
		return false, err
	}
	if found {
		s.removeStoredFiles(fileIDs)
	}
	return found, nil
}

func (s *AuthService) purgeAccount(ctx context.Context, kullaniciID bson.ObjectID) ([]string, error) {
	user, err := s.users.GetByID(ctx, kullaniciID)
	if err != nil {
		return nil, err
	}
	if baska, err := s.ownerHasOtherMembers(ctx, user); err != nil {
		return nil, err
	} else if baska {
		return nil, auth.ErrTransferOwnership
	}
	var fileIDs []string
	if !user.OrganizationID.IsZero() {
		n, err := s.users.CountByOrg(ctx, user.OrganizationID)
		if err != nil {
			return nil, err
		}
		if user.Role == repository.RoleOwner || n <= 1 {
			ids, err := s.storedFilesByOrg(ctx, user.OrganizationID)
			if err != nil {
				return nil, err
			}
			fileIDs = ids
			if err := s.deleteOrg(ctx, user.OrganizationID, false); err != nil {
				return nil, err
			}
		} else if err := s.users.DetachOrg(ctx, kullaniciID); err != nil {
			return nil, err
		}
	} else {
		ids, err := s.storedFilesByUser(ctx, kullaniciID)
		if err != nil {
			return nil, err
		}
		fileIDs = ids
		if err := s.soz.DeleteByUser(ctx, kullaniciID); err != nil {
			return nil, err
		}
	}
	if err := s.tokens.DeleteByUser(ctx, kullaniciID); err != nil {
		return nil, err
	}
	if err := s.mfa.DeleteByUser(ctx, kullaniciID); err != nil {
		return nil, err
	}
	if err := s.sessions.DeleteByUser(ctx, kullaniciID); err != nil {
		return nil, err
	}
	if err := s.devices.DeleteByUser(ctx, kullaniciID); err != nil {
		return nil, err
	}
	if err := s.audit.AnonymizeByUser(ctx, kullaniciID); err != nil {
		return nil, err
	}
	if s.deleteFailAt == deleteStepUser {
		return nil, errDeleteProbe
	}
	if err := s.users.Delete(ctx, kullaniciID); err != nil {
		return nil, err
	}
	if err := s.audit.Insert(ctx, &repository.AuditRecord{
		UserID:     repository.UserDeleted,
		Event:      repository.EventAccountDeleted,
		IPAddress:  "",
		UserAgent:  "",
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		return nil, err
	}
	return fileIDs, nil
}

// VerilerimiIndir kullanıcının dışa aktarılabilir verisini JSON döner. Hash ve token yok.
func (s *AuthService) ExportData(ctx context.Context) (string, error) {
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
	if !user.OrganizationID.IsZero() {
		orgSoz, err := s.soz.ListByOrg(ctx, user.OrganizationID)
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
		"eposta":           user.Email,
		"durum":            user.Status,
		"epostaDogrulandi": user.EmailVerified,
		"olusturmaTarihi":  user.CreatedAt,
		"cihazlar":         exportDevices(cihazlar),
		"oturumlar":        exportSessions(oturumlar),
		"sozlesmeler":      exportContracts(sozlesmeler),
		"denetim":          exportAudit(denetim),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		log.Printf("veri dışa aktarma başarısız: %v", err)
		return "", repository.ErrStore
	}
	return string(raw), nil
}

func exportDevices(docs []repository.Device) []map[string]any {
	out := make([]map[string]any, 0, len(docs))
	for i := range docs {
		d := docs[i]
		out = append(out, map[string]any{
			"id":             d.ID.Hex(),
			"ad":             d.Name,
			"guvenilir":      d.Trusted,
			"ilkGorulme":     d.FirstSeen,
			"sonGorulme":     d.LastSeen,
			"ipAdresi":       d.IPAddress,
			"kullaniciAjani": d.UserAgent,
		})
	}
	return out
}

func exportSessions(docs []repository.Session) []map[string]any {
	out := make([]map[string]any, 0, len(docs))
	for i := range docs {
		d := docs[i]
		item := map[string]any{
			"id":              d.ID.Hex(),
			"olusturmaTarihi": d.CreatedAt,
			"sonKullanma":     d.ExpiresAt,
			"iptalEdildi":     d.Revoked,
			"ipAdresi":        d.IPAddress,
			"kullaniciAjani":  d.UserAgent,
		}
		if !d.DeviceID.IsZero() {
			item["cihazId"] = d.DeviceID.Hex()
		}
		out = append(out, item)
	}
	return out
}

func exportContracts(docs []repository.Contract) []map[string]any {
	out := make([]map[string]any, 0, len(docs))
	for i := range docs {
		d := docs[i]
		item := map[string]any{
			"id":               d.ID.Hex(),
			"olusturmaTarihi":  d.CreatedAt,
			"guncellemeTarihi": d.UpdatedAt,
			"durum":            d.Status,
		}
		if d.FileName != nil {
			item["dosyaAdi"] = *d.FileName
		}
		out = append(out, item)
	}
	return out
}

func exportAudit(docs []repository.AuditRecord) []map[string]any {
	out := make([]map[string]any, 0, len(docs))
	for i := range docs {
		d := docs[i]
		out = append(out, map[string]any{
			"olay":  d.Event,
			"zaman": d.OccurredAt,
			"detay": d.Detail,
		})
	}
	return out
}
