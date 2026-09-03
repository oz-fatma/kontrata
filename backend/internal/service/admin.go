package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/agent"
	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const defaultPromptID = "varsayilan"

var (
	ErrEmptyPrompt     = errors.New("prompt metni boş")
	ErrInvalidSettings = errors.New("ayar değerleri geçersiz")
)

func (s *AuthService) AttachOrgLLM(prompts *repository.PromptVersionRepository, settings *repository.OrgSettingsRepository) {
	if s == nil {
		return
	}
	s.prompts = prompts
	s.settings = settings
}

func (s *AuthService) AttachLLMCalls(calls *repository.LLMCallRepository) {
	if s == nil {
		return
	}
	s.llmCalls = calls
}

func (s *AuthService) requireOwnerOrg(ctx context.Context) (actor, error) {
	act, err := s.actor(ctx)
	if err != nil {
		return actor{}, err
	}
	if !act.can(opPromptManage) || !act.hasOrg() {
		return actor{}, auth.ErrForbidden
	}
	return act, nil
}

func (s *AuthService) requirePromptAdmin(ctx context.Context) (actor, error) {
	act, err := s.requireOwnerOrg(ctx)
	if err != nil {
		return actor{}, err
	}
	if s.prompts == nil || s.settings == nil {
		return actor{}, repository.ErrUnavailable
	}
	return act, nil
}

func (s *AuthService) PromptVersions(ctx context.Context, tip model.PromptTipi) ([]*model.PromptSurumu, error) {
	act, err := s.requirePromptAdmin(ctx)
	if err != nil {
		return nil, err
	}
	kind := string(tip)
	docs, err := s.prompts.ListByKind(ctx, act.orgID(), kind)
	if err != nil {
		return nil, err
	}
	out := make([]*model.PromptSurumu, 0, len(docs))
	for i := range docs {
		out = append(out, promptToModel(&docs[i]))
	}
	return out, nil
}

func (s *AuthService) ActivePrompt(ctx context.Context, tip model.PromptTipi) (*model.PromptSurumu, error) {
	act, err := s.requirePromptAdmin(ctx)
	if err != nil {
		return nil, err
	}
	kind := string(tip)
	doc, err := s.prompts.GetActive(ctx, act.orgID(), kind)
	if err == nil {
		return promptToModel(doc), nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	return defaultPromptModel(tip), nil
}

func (s *AuthService) UpdatePrompt(ctx context.Context, tip model.PromptTipi, icerik string) (*model.PromptSurumu, error) {
	act, err := s.requirePromptAdmin(ctx)
	if err != nil {
		return nil, err
	}
	icerik = strings.TrimSpace(icerik)
	if icerik == "" {
		return nil, ErrEmptyPrompt
	}
	kind := string(tip)
	max, err := s.prompts.MaxVersion(ctx, act.orgID(), kind)
	if err != nil {
		return nil, err
	}
	if err := s.prompts.DeactivateKind(ctx, act.orgID(), kind); err != nil {
		return nil, err
	}
	doc := repository.PromptVersion{
		OrgID:     act.orgID(),
		Kind:      kind,
		Content:   icerik,
		Version:   max + 1,
		CreatedBy: act.user.ID,
		CreatedAt: time.Now().UTC(),
		Active:    true,
	}
	if err := s.prompts.Insert(ctx, &doc); err != nil {
		return nil, err
	}
	s.writeAudit(ctx, &act.user.ID, repository.EventPromptUpdated, fmt.Sprintf("tip=%s surum=%d", kind, doc.Version))
	return promptToModel(&doc), nil
}

func (s *AuthService) RevertPrompt(ctx context.Context, id string) (*model.PromptSurumu, error) {
	act, err := s.requirePromptAdmin(ctx)
	if err != nil {
		return nil, err
	}
	oid, err := bson.ObjectIDFromHex(strings.TrimSpace(id))
	if err != nil {
		return nil, repository.ErrNotFound
	}
	doc, err := s.prompts.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	if doc.OrgID != act.orgID() {
		return nil, repository.ErrNotFound
	}
	if doc.Active {
		return promptToModel(doc), nil
	}
	if err := s.prompts.DeactivateKind(ctx, act.orgID(), doc.Kind); err != nil {
		return nil, err
	}
	if err := s.prompts.Activate(ctx, doc.ID); err != nil {
		return nil, err
	}
	doc.Active = true
	s.writeAudit(ctx, &act.user.ID, repository.EventPromptReverted, fmt.Sprintf("tip=%s surum=%d", doc.Kind, doc.Version))
	return promptToModel(doc), nil
}

func (s *AuthService) Settings(ctx context.Context) (*model.Ayarlar, error) {
	act, err := s.requirePromptAdmin(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := s.settings.GetByOrg(ctx, act.orgID())
	if errors.Is(err, repository.ErrNotFound) {
		d := repository.DefaultOrgSettings(act.orgID())
		if upErr := s.settings.Upsert(ctx, &d); upErr != nil {
			return nil, upErr
		}
		return settingsToModel(&d), nil
	}
	if err != nil {
		return nil, err
	}
	return settingsToModel(doc), nil
}

func (s *AuthService) UpdateSettings(ctx context.Context, risk *float64, maxToken *int32) (*model.Ayarlar, error) {
	act, err := s.requirePromptAdmin(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := s.settings.GetByOrg(ctx, act.orgID())
	if errors.Is(err, repository.ErrNotFound) {
		d := repository.DefaultOrgSettings(act.orgID())
		doc = &d
		err = nil
	}
	if err != nil {
		return nil, err
	}
	var changed []string
	if risk != nil {
		if *risk < 0 || *risk > 1 {
			return nil, ErrInvalidSettings
		}
		if doc.RiskThreshold != *risk {
			changed = append(changed, "denetciRiskEsigi")
		}
		doc.RiskThreshold = *risk
	}
	if maxToken != nil {
		if *maxToken < 1 || *maxToken > 8192 {
			return nil, ErrInvalidSettings
		}
		if doc.MaxToken != *maxToken {
			changed = append(changed, "maxToken")
		}
		doc.MaxToken = *maxToken
	}
	if len(changed) == 0 {
		return settingsToModel(doc), nil
	}
	doc.UpdatedAt = time.Now().UTC()
	doc.UpdatedBy = act.user.ID
	if err := s.settings.Upsert(ctx, doc); err != nil {
		return nil, err
	}
	s.writeAudit(ctx, &act.user.ID, repository.EventSettingsUpdated, "alan="+strings.Join(changed, ","))
	return settingsToModel(doc), nil
}

func promptToModel(doc *repository.PromptVersion) *model.PromptSurumu {
	if doc == nil {
		return nil
	}
	created := doc.CreatedAt
	if created.IsZero() {
		created = time.Unix(0, 0).UTC()
	}
	return &model.PromptSurumu{
		ID:                   doc.ID.Hex(),
		Tip:                  model.PromptTipi(doc.Kind),
		Icerik:               doc.Content,
		Surum:                doc.Version,
		Aktif:                doc.Active,
		OlusturmaTarihi:      created,
		OlusturanKullaniciID: doc.CreatedBy.Hex(),
	}
}

func defaultPromptModel(tip model.PromptTipi) *model.PromptSurumu {
	icerik := agent.SYSTEM_PROMPT
	if tip == model.PromptTipiDenetci {
		icerik = agent.AUDITOR_SYSTEM_PROMPT
	}
	return &model.PromptSurumu{
		ID:                   defaultPromptID,
		Tip:                  tip,
		Icerik:               icerik,
		Surum:                0,
		Aktif:                true,
		OlusturmaTarihi:      time.Unix(0, 0).UTC(),
		OlusturanKullaniciID: defaultPromptID,
	}
}

func settingsToModel(doc *repository.OrgSettings) *model.Ayarlar {
	if doc == nil {
		d := repository.DefaultOrgSettings(bson.ObjectID{})
		doc = &d
	}
	updated := doc.UpdatedAt
	if updated.IsZero() {
		updated = time.Unix(0, 0).UTC()
	}
	out := &model.Ayarlar{
		DenetciRiskEsigi: doc.RiskThreshold,
		MaxToken:         doc.MaxToken,
		GuncellemeTarihi: updated,
	}
	if !doc.UpdatedBy.IsZero() {
		id := doc.UpdatedBy.Hex()
		out.GuncelleyenKullaniciID = &id
	}
	return out
}

func (s *AuthService) deleteOrgLLM(ctx context.Context, orgID bson.ObjectID) error {
	if s.prompts != nil {
		if err := s.prompts.DeleteByOrg(ctx, orgID); err != nil {
			return err
		}
	}
	if s.settings != nil {
		if err := s.settings.DeleteByOrg(ctx, orgID); err != nil {
			return err
		}
	}
	if s.llmCalls != nil {
		if err := s.llmCalls.DeleteByOrg(ctx, orgID); err != nil {
			return err
		}
	}
	return nil
}
