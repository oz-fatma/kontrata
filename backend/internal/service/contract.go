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

// ContractService sözleşme iş kurallarını taşır.
type ContractService struct {
	repo  *repository.ContractRepository
	users *repository.UserRepository
}

func NewContractService(repo *repository.ContractRepository, users *repository.UserRepository) *ContractService {
	return &ContractService{repo: repo, users: users}
}

func (s *ContractService) Create(ctx context.Context, girdi model.SozlesmeGirdi) (*model.Sozlesme, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opContractWrite) {
		return nil, auth.ErrForbidden
	}
	doc := fromInput(girdi)
	now := time.Now().UTC()
	doc.CreatedAt = now
	doc.UpdatedAt = now
	if doc.Status == "" {
		doc.Status = string(model.SozlesmeDurumuYuklendi)
	}
	doc.UserID = act.user.ID
	doc.OrganizationID = act.orgID()
	if err := s.repo.Create(ctx, &doc); err != nil {
		log.Printf("sozlesme oluşturma başarısız: %v", err)
		return nil, err
	}
	return toModel(&doc), nil
}

func (s *ContractService) Get(ctx context.Context, id string) (*model.Sozlesme, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opContractRead) {
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
	if !act.ownsContract(doc) {
		return nil, nil
	}
	return toModel(doc), nil
}

func (s *ContractService) List(ctx context.Context, limit, offset *int32) ([]*model.Sozlesme, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opContractRead) {
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
	docs, err := s.repo.List(ctx, act.contractFilter(), int64(l), int64(o))
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

func (s *ContractService) Update(ctx context.Context, id string, girdi model.SozlesmeGirdi) (*model.Sozlesme, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opContractWrite) {
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
	if !act.ownsContract(existing) {
		return nil, repository.ErrNotFound
	}
	doc := fromInput(girdi)
	doc.ID = existing.ID
	doc.UserID = existing.UserID
	doc.OrganizationID = existing.OrganizationID
	doc.CreatedAt = existing.CreatedAt
	doc.UpdatedAt = time.Now().UTC()
	if doc.Status == "" {
		doc.Status = existing.Status
	}
	if err := s.repo.Update(ctx, &doc); err != nil {
		log.Printf("sozlesme güncelleme başarısız: %v", err)
		return nil, err
	}
	return toModel(&doc), nil
}

func (s *ContractService) Delete(ctx context.Context, id string) (bool, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return false, err
	}
	if !act.can(opContractDelete) {
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
	if !act.ownsContract(existing) {
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
