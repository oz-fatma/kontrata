package service

import (
	"context"
	"log"
	"time"

	"github.com/oz-fatma/kontrata/backend/internal/llm"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type llmRecorder struct {
	repo *repository.LLMCallRepository
}

func NewLLMRecorder(repo *repository.LLMCallRepository) llm.Recorder {
	if repo == nil {
		return llm.NopRecorder{}
	}
	return &llmRecorder{repo: repo}
}

func (r *llmRecorder) Record(_ context.Context, rec llm.CallRecord) {
	if r == nil || r.repo == nil {
		return
	}
	if rec.Start.IsZero() {
		rec.Start = time.Now().UTC()
	}
	if rec.End.IsZero() {
		rec.End = rec.Start
	}
	if rec.DurationMs < 0 {
		rec.DurationMs = 0
	}
	doc := repository.LLMCall{
		Agent:         rec.Agent,
		Endpoint:      rec.Endpoint,
		Start:         rec.Start,
		End:           rec.End,
		DurationMs:    rec.DurationMs,
		InChars:       rec.InChars,
		OutChars:      rec.OutChars,
		Success:       rec.Success,
		ErrorType:     rec.ErrorType,
		Attempt:       int32(rec.Attempt),
		PromptVersion: rec.PromptVersion,
	}
	if oid, err := bson.ObjectIDFromHex(rec.OrgID); err == nil {
		doc.OrgID = oid
	}
	if oid, err := bson.ObjectIDFromHex(rec.ContractID); err == nil {
		doc.ContractID = oid
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := r.repo.Insert(ctx, &doc); err != nil {
			log.Printf("llm izleme yazilamadi: %v", err)
		}
	}()
}
