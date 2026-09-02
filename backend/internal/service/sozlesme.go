package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

// SozlesmeService sözleşme iş kurallarını taşır.
type SozlesmeService struct {
	repo *repository.SozlesmeRepository
}

func NewSozlesmeService(repo *repository.SozlesmeRepository) *SozlesmeService {
	return &SozlesmeService{repo: repo}
}

func (s *SozlesmeService) Create(ctx context.Context, girdi model.SozlesmeGirdi) (*model.Sozlesme, error) {
	doc := fromGirdi(girdi)
	now := time.Now().UTC()
	doc.OlusturmaTarihi = now
	doc.GuncellemeTarihi = now
	if doc.Durum == "" {
		doc.Durum = string(model.SozlesmeDurumuYuklendi)
	}
	if err := s.repo.Create(ctx, &doc); err != nil {
		log.Printf("sozlesme oluşturma başarısız")
		return nil, err
	}
	return toModel(&doc), nil
}

func (s *SozlesmeService) Get(ctx context.Context, id string) (*model.Sozlesme, error) {
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		if errors.Is(err, repository.ErrInvalidID) {
			return nil, err
		}
		log.Printf("sozlesme okuma başarısız")
		return nil, err
	}
	return toModel(doc), nil
}

func (s *SozlesmeService) List(ctx context.Context, limit, offset *int32) ([]*model.Sozlesme, error) {
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
	docs, err := s.repo.List(ctx, int64(l), int64(o))
	if err != nil {
		log.Printf("sozlesme listeleme başarısız")
		return nil, err
	}
	out := make([]*model.Sozlesme, 0, len(docs))
	for i := range docs {
		out = append(out, toModel(&docs[i]))
	}
	return out, nil
}

func (s *SozlesmeService) Update(ctx context.Context, id string, girdi model.SozlesmeGirdi) (*model.Sozlesme, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrInvalidID) {
			return nil, err
		}
		log.Printf("sozlesme güncelleme (okuma) başarısız")
		return nil, err
	}
	doc := fromGirdi(girdi)
	doc.ID = existing.ID
	doc.OlusturmaTarihi = existing.OlusturmaTarihi
	doc.GuncellemeTarihi = time.Now().UTC()
	if doc.Durum == "" {
		doc.Durum = existing.Durum
	}
	if err := s.repo.Update(ctx, &doc); err != nil {
		log.Printf("sozlesme güncelleme başarısız")
		return nil, err
	}
	return toModel(&doc), nil
}

func (s *SozlesmeService) Delete(ctx context.Context, id string) (bool, error) {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrInvalidID) {
			return false, err
		}
		log.Printf("sozlesme silme başarısız")
		return false, err
	}
	log.Printf("denetim: sozlesme silindi")
	return true, nil
}
