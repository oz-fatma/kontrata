package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

// SozlesmeService sözleşme iş kurallarını taşır.
type SozlesmeService struct {
	repo  *repository.SozlesmeRepository
	users *repository.KullaniciRepository
}

func NewSozlesmeService(repo *repository.SozlesmeRepository, users *repository.KullaniciRepository) *SozlesmeService {
	return &SozlesmeService{repo: repo, users: users}
}

func (s *SozlesmeService) Create(ctx context.Context, girdi model.SozlesmeGirdi) (*model.Sozlesme, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opSozlesmeYaz) {
		return nil, auth.ErrForbidden
	}
	doc := fromGirdi(girdi)
	now := time.Now().UTC()
	doc.OlusturmaTarihi = now
	doc.GuncellemeTarihi = now
	if doc.Durum == "" {
		doc.Durum = string(model.SozlesmeDurumuYuklendi)
	}
	doc.KullaniciID = act.user.ID
	doc.OrganizasyonID = act.orgID()
	if err := s.repo.Create(ctx, &doc); err != nil {
		log.Printf("sozlesme oluşturma başarısız: %v", err)
		return nil, err
	}
	return toModel(&doc), nil
}

func (s *SozlesmeService) Get(ctx context.Context, id string) (*model.Sozlesme, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opSozlesmeOku) {
		return nil, auth.ErrForbidden
	}
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		if errors.Is(err, repository.ErrInvalidID) {
			return nil, err
		}
		log.Printf("sozlesme okuma başarısız: %v", err)
		return nil, err
	}
	if !act.ownsSozlesme(doc) {
		return nil, nil
	}
	return toModel(doc), nil
}

func (s *SozlesmeService) List(ctx context.Context, limit, offset *int32) ([]*model.Sozlesme, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opSozlesmeOku) {
		return nil, auth.ErrForbidden
	}
	l, o := defaultListLimit, 0
	if limit != nil {
		l = int(*limit)
	}
	if offset != nil {
		o = int(*offset)
	}
	if l <= 0 {
		l = defaultListLimit
	}
	if l > maxListLimit {
		l = maxListLimit
	}
	if o < 0 {
		o = 0
	}
	docs, err := s.repo.List(ctx, act.sozlesmeFilter(), int64(l), int64(o))
	if err != nil {
		log.Printf("sozlesme listeleme başarısız: %v", err)
		return nil, err
	}
	out := make([]*model.Sozlesme, 0, len(docs))
	for i := range docs {
		out = append(out, toModel(&docs[i]))
	}
	return out, nil
}

func (s *SozlesmeService) Update(ctx context.Context, id string, girdi model.SozlesmeGirdi) (*model.Sozlesme, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opSozlesmeYaz) {
		return nil, auth.ErrForbidden
	}
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrInvalidID) {
			return nil, err
		}
		log.Printf("sozlesme güncelleme (okuma) başarısız: %v", err)
		return nil, err
	}
	if !act.ownsSozlesme(existing) {
		return nil, repository.ErrNotFound
	}
	doc := fromGirdi(girdi)
	doc.ID = existing.ID
	doc.KullaniciID = existing.KullaniciID
	doc.OrganizasyonID = existing.OrganizasyonID
	doc.OlusturmaTarihi = existing.OlusturmaTarihi
	doc.GuncellemeTarihi = time.Now().UTC()
	if doc.Durum == "" {
		doc.Durum = existing.Durum
	}
	if err := s.repo.Update(ctx, &doc); err != nil {
		log.Printf("sozlesme güncelleme başarısız: %v", err)
		return nil, err
	}
	return toModel(&doc), nil
}

func (s *SozlesmeService) Delete(ctx context.Context, id string) (bool, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return false, err
	}
	if !act.can(opSozlesmeSil) {
		return false, auth.ErrForbidden
	}
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrInvalidID) {
			return false, err
		}
		log.Printf("sozlesme silme başarısız: %v", err)
		return false, err
	}
	if !act.ownsSozlesme(existing) {
		return false, repository.ErrNotFound
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrInvalidID) {
			return false, err
		}
		log.Printf("sozlesme silme başarısız: %v", err)
		return false, err
	}
	log.Printf("denetim: sozlesme silindi")
	return true, nil
}
