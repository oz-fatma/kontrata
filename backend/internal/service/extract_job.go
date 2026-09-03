package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/agent"
	"github.com/oz-fatma/kontrata/backend/internal/llm"
	"github.com/oz-fatma/kontrata/backend/internal/pdf"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const extractJobTimeout = 12 * time.Minute

func (s *ContractService) enqueueExtract(id string) {
	if s == nil || s.jobs == nil || id == "" {
		return
	}
	select {
	case s.jobs <- id:
	default:
		go func() {
			s.jobs <- id
		}()
	}
}

// StartExtractWorker kuyruktan iş çekip çıkarımı arka planda yürütür.
func (s *ContractService) StartExtractWorker(ctx context.Context) {
	if s == nil || s.jobs == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case id := <-s.jobs:
				s.runExtract(id)
			}
		}
	}()
}

func (s *ContractService) runExtract(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), extractJobTimeout)
	defer cancel()

	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Printf("cikarma kayit okunamadi: %v", err)
		return
	}
	doc.Status = string(model.SozlesmeDurumuIsleniyor)
	doc.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, doc); err != nil {
		log.Printf("cikarma durum guncellenemedi: %v", err)
		return
	}

	pages, err := s.readPages(doc.StoredFileID)
	if err != nil {
		log.Printf("cikarma pdf hatasi: %v", err)
		s.failExtract(ctx, doc, []string{userExtractError(err)}, nil)
		return
	}
	log.Printf("cikarma basladi sayfa=%d", len(pages))

	reader := &agent.Reader{LLM: s.llm, DumpDir: s.dump, ContractID: id}
	res, err := reader.Extract(ctx, pages)
	if err != nil {
		log.Printf("cikarma model hatasi: %v", err)
		msg := "model yanıt vermedi"
		if errors.Is(err, llm.ErrColdStart) {
			msg = err.Error()
		}
		s.failExtract(ctx, doc, []string{msg}, res)
		return
	}
	applyExtract(doc, res)
	doc.Findings = nil
	doc.AuditorSeconds = nil
	s.runAudit(ctx, doc, res.Data, pages)
	if len(res.SchemaErrors) > 0 {
		doc.Status = string(model.SozlesmeDurumuHata)
	} else {
		doc.Status = string(model.SozlesmeDurumuIncelenmeyiBekliyor)
	}
	doc.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, doc); err != nil {
		log.Printf("cikarma kayit yazilamadi: %v", err)
		return
	}
	log.Printf("cikarma bitti durum=%s sure=%s duzeltme=%d hata=%d deneme=%d bulgu=%d",
		doc.Status, res.Duration, len(res.Repairs), len(res.SchemaErrors), res.RetryCount, len(doc.Findings))
}

func (s *ContractService) readPages(storedID string) ([]string, error) {
	if s.files == nil || storedID == "" {
		return nil, pdf.ErrUnreadable
	}
	f, err := s.files.Open(storedID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("cikarma dosya kapatilamadi: %v", err)
		}
	}()
	return pdf.ExtractText(f)
}

func (s *ContractService) runAudit(ctx context.Context, doc *repository.Contract, data map[string]any, pages []string) {
	if doc == nil {
		return
	}
	a := &agent.Auditor{LLM: llm.LimitTokens(s.llm, agent.AuditorMaxTokens)}
	res, err := a.Audit(ctx, data, pages)
	applyAuditOutcome(doc, res, err)
	n := 0
	if res != nil {
		n = len(res.Findings)
	}
	log.Printf("denetci kaydedildi bulgu=%d", n)
}

func (s *ContractService) failExtract(ctx context.Context, doc *repository.Contract, errs []string, res *agent.ExtractResult) {
	if res != nil {
		applyExtract(doc, res)
	}
	doc.SchemaErrors = append(doc.SchemaErrors, errs...)
	doc.Status = string(model.SozlesmeDurumuHata)
	doc.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, doc); err != nil {
		log.Printf("cikarma hata kaydi yazilamadi: %v", err)
	}
}

func userExtractError(err error) string {
	switch {
	case errors.Is(err, pdf.ErrNoTextLayer):
		return "PDF'de metin katmanı yok"
	case errors.Is(err, pdf.ErrEmpty):
		return "PDF boş"
	case errors.Is(err, pdf.ErrUnreadable):
		return "PDF okunamadı"
	default:
		return "çıkarım başarısız"
	}
}
