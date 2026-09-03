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

var (
	// ErrApprovedReadOnly onaylı sözleşmenin değiştirilmesini reddeder.
	ErrApprovedReadOnly = errors.New("onaylı sözleşme düzenlenemez")
	// ErrNotAwaitingReview onayın yalnızca INCELENMEYI_BEKLIYOR durumunda olduğunu söyler.
	ErrNotAwaitingReview = errors.New("yalnızca incelenmeyi bekleyen sözleşme onaylanır")
	// ErrUnknownField tanınmayan alan yoludur.
	ErrUnknownField = errors.New("bilinmeyen alan")
	// ErrInvalidFieldValue alan değerinin ayrıştırılamadığını söyler.
	ErrInvalidFieldValue = errors.New("alan değeri okunamadı")
)

// ContractService sözleşme iş kurallarını taşır.
type ContractService struct {
	repo     *repository.ContractRepository
	users    *repository.UserRepository
	files    *filestore.Store
	llm      llm.Client
	audit    *repository.AuditRepository
	jobs     chan string
	dump     string
	prompts  *repository.PromptVersionRepository
	settings *repository.OrgSettingsRepository
}

func NewContractService(repo *repository.ContractRepository, users *repository.UserRepository) *ContractService {
	return &ContractService{
		repo:  repo,
		users: users,
		jobs:  make(chan string, extractQueueSize),
	}
}

// AttachExtract çıkarım işçisi ve dosya deposunu bağlar.
// dumpDir doluysa modele giden (maskelenmiş) metin ve ham çıktı oraya yazılır (LLM_DEBUG_DUMP).
func (s *ContractService) AttachExtract(files *filestore.Store, client llm.Client, dumpDir string) {
	s.files = files
	s.llm = client
	s.dump = dumpDir
}

// AttachAudit denetim kaydı deposunu bağlar.
func (s *ContractService) AttachAudit(audit *repository.AuditRepository) {
	s.audit = audit
}

// AttachOrgLLM organizasyon prompt ve ayar depolarını bağlar.
func (s *ContractService) AttachOrgLLM(prompts *repository.PromptVersionRepository, settings *repository.OrgSettingsRepository) {
	if s == nil {
		return
	}
	s.prompts = prompts
	s.settings = settings
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
	if existing.Status == string(model.SozlesmeDurumuOnaylandi) {
		return nil, ErrApprovedReadOnly
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

// Approve INCELENMEYI_BEKLIYOR sözleşmesini ONAYLANDI yapar.
func (s *ContractService) Approve(ctx context.Context, id string) (*model.Sozlesme, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opContractApprove) {
		return nil, auth.ErrForbidden
	}
	doc, err := s.ownedContract(ctx, act, id)
	if err != nil {
		return nil, err
	}
	if doc.Status != string(model.SozlesmeDurumuIncelenmeyiBekliyor) {
		return nil, ErrNotAwaitingReview
	}
	doc.Status = string(model.SozlesmeDurumuOnaylandi)
	doc.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, doc); err != nil {
		log.Printf("sozlesme onaylama başarısız: %v", err)
		return nil, err
	}
	s.writeAudit(ctx, act.user.ID, repository.EventContractApproved, "")
	return toModel(doc), nil
}

// UpdateField tek bir çıkarılmış alanı yazar, güveni 1.0 yapar ve Denetçi'yi yeniden çalıştırır.
func (s *ContractService) UpdateField(ctx context.Context, id, path string, value any) (*model.Sozlesme, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !act.can(opContractWrite) {
		return nil, auth.ErrForbidden
	}
	doc, err := s.ownedContract(ctx, act, id)
	if err != nil {
		return nil, err
	}
	if doc.Status == string(model.SozlesmeDurumuOnaylandi) {
		return nil, ErrApprovedReadOnly
	}
	normalized, err := applyFieldValue(doc, path, value)
	if err != nil {
		return nil, err
	}
	markFieldManuallyFixed(doc, normalized)
	pages, _ := s.readPages(doc.StoredFileID)
	s.runAudit(ctx, doc, auditDataFromContract(doc), pages)
	doc.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, doc); err != nil {
		log.Printf("sozlesme alan güncelleme başarısız: %v", err)
		return nil, err
	}
	s.writeAudit(ctx, act.user.ID, repository.EventContractFieldUpdated, normalized)
	return toModel(doc), nil
}

func (s *ContractService) ownedContract(ctx context.Context, act actor, id string) (*repository.Contract, error) {
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrInvalidID) {
			return nil, err
		}
		log.Printf("sozlesme okuma başarısız: %v", err)
		return nil, err
	}
	if !act.ownsContract(doc) {
		return nil, repository.ErrNotFound
	}
	return doc, nil
}

func (s *ContractService) writeAudit(ctx context.Context, userID any, event, detail string) {
	if s == nil || s.audit == nil {
		return
	}
	meta := auth.MetaFrom(ctx)
	rec := repository.AuditRecord{
		UserID:     userID,
		Event:      event,
		IPAddress:  meta.IP,
		UserAgent:  meta.UserAgent,
		OccurredAt: time.Now().UTC(),
		Detail:     detail,
	}
	if err := s.audit.Insert(ctx, &rec); err != nil {
		log.Printf("denetim kaydı yazılamadı: %v", err)
	}
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

// RunExtractForTest çıkarımı senkron çalıştırır; yalnızca testler kullanır.
func (s *ContractService) RunExtractForTest(id string) {
	s.runExtract(id)
}

func sanitizeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "sozlesme.pdf"
	}
	return name
}
