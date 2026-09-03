package service

import (
	"context"
	"errors"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/filestore"
	"github.com/oz-fatma/kontrata/backend/internal/llm"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
	extractQueueSize = 16
)

// ContractService sözleşme iş kurallarını taşır.
type ContractService struct {
	repo  *repository.ContractRepository
	users *repository.UserRepository
	files *filestore.Store
	llm   llm.Client
	jobs  chan string
	dump  string
}

func NewContractService(repo *repository.ContractRepository, users *repository.UserRepository) *ContractService {
	return &ContractService{
		repo:  repo,
		users: users,
		jobs:  make(chan string, extractQueueSize),
	}
}

// AttachExtract çıkarım işçisi ve dosya deposunu bağlar.
// dumpDir doluysa ham model çıktısı oraya yazılır (LLM_DEBUG_DUMP).
func (s *ContractService) AttachExtract(files *filestore.Store, client llm.Client, dumpDir string) {
	s.files = files
	s.llm = client
	s.dump = dumpDir
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
	doc.StoredFileID = existing.StoredFileID
	if doc.FileName == nil {
		doc.FileName = existing.FileName
	}
	if len(doc.Repairs) == 0 {
		doc.Repairs = existing.Repairs
	}
	if len(doc.SchemaErrors) == 0 {
		doc.SchemaErrors = existing.SchemaErrors
	}
	if doc.ProcessingSeconds == nil {
		doc.ProcessingSeconds = existing.ProcessingSeconds
	}
	if len(doc.Findings) == 0 {
		doc.Findings = existing.Findings
	}
	if doc.AuditorSeconds == nil {
		doc.AuditorSeconds = existing.AuditorSeconds
	}
	if len(doc.ExtractionMeta) == 0 {
		doc.ExtractionMeta = existing.ExtractionMeta
	}
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
	stored := existing.StoredFileID
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrInvalidID) {
			return false, err
		}
		log.Printf("sozlesme silme başarısız: %v", err)
		return false, err
	}
	if s.files != nil && stored != "" {
		if err := s.files.Remove(stored); err != nil {
			log.Printf("sozlesme dosyası silinemedi: %v", err)
		}
	}
	log.Printf("denetim: sozlesme silindi")
	return true, nil
}

// Upload PDF'i diske yazar, YUKLENDI kaydı oluşturur ve çıkarımı kuyruğa atar.
func (s *ContractService) Upload(ctx context.Context, filename string, r io.Reader) (*model.Sozlesme, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opContractWrite) {
		return nil, auth.ErrForbidden
	}
	if s.files == nil {
		return nil, filestore.ErrNotFound
	}
	name := sanitizeFileName(filename)
	id, err := s.files.Save(r)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	doc := repository.Contract{
		CreatedAt:      now,
		UpdatedAt:      now,
		Status:         string(model.SozlesmeDurumuYuklendi),
		FileName:       &name,
		StoredFileID:   id,
		UserID:         act.user.ID,
		OrganizationID: act.orgID(),
		RoomAllotments: []repository.RoomAllotment{},
		Prices:         []repository.Price{},
		StopSale:       []repository.StopSaleRange{},
	}
	if err := s.repo.Create(ctx, &doc); err != nil {
		_ = s.files.Remove(id)
		log.Printf("sozlesme oluşturma başarısız: %v", err)
		return nil, err
	}
	s.enqueueExtract(doc.ID.Hex())
	log.Printf("sozlesme yüklendi")
	return toModel(&doc), nil
}

func sanitizeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "sozlesme.pdf"
	}
	return name
}
