package service

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/oz-fatma/kontrata/backend/graph/model"
	"github.com/oz-fatma/kontrata/backend/internal/llm"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

const defaultMetricsHours = 24

func (s *AuthService) LLMMetrics(ctx context.Context, sonSaat *int32, baslangic *time.Time) (*model.LlmMetrik, error) {
	act, err := s.requireOwnerOrg(ctx)
	if err != nil {
		return nil, err
	}
	if s.llmCalls == nil {
		return nil, repository.ErrUnavailable
	}
	since := metricsSince(time.Now().UTC(), sonSaat, baslangic)
	rows, err := s.llmCalls.ListSince(ctx, act.orgID(), since)
	if err != nil {
		return nil, err
	}
	return buildLLMMetrics(rows), nil
}

func metricsSince(now time.Time, sonSaat *int32, baslangic *time.Time) time.Time {
	if baslangic != nil && !baslangic.IsZero() {
		return baslangic.UTC()
	}
	hours := defaultMetricsHours
	if sonSaat != nil && *sonSaat > 0 {
		hours = int(*sonSaat)
		if hours > 168 {
			hours = 168
		}
	}
	return now.UTC().Add(-time.Duration(hours) * time.Hour)
}

func (s *AuthService) LLMCalls(ctx context.Context, limit *int32) ([]*model.LlmCagri, error) {
	act, err := s.requireOwnerOrg(ctx)
	if err != nil {
		return nil, err
	}
	if s.llmCalls == nil {
		return nil, repository.ErrUnavailable
	}
	n := int64(20)
	if limit != nil && *limit > 0 {
		n = int64(*limit)
		if n > 100 {
			n = 100
		}
	}
	rows, err := s.llmCalls.ListRecent(ctx, act.orgID(), n)
	if err != nil {
		return nil, err
	}
	out := make([]*model.LlmCagri, 0, len(rows))
	for i := range rows {
		out = append(out, llmCallToModel(&rows[i]))
	}
	return out, nil
}

func buildLLMMetrics(rows []repository.LLMCall) *model.LlmMetrik {
	out := &model.LlmMetrik{
		AgentBazinda: []*model.LlmAgentMetrik{},
		UcBazinda:    []*model.LlmUcMetrik{},
		HataDagilimi: []*model.LlmHataAdet{},
	}
	if len(rows) == 0 {
		return out
	}
	type agg struct {
		n, ok int
		sum   int64
	}
	byAgent := map[string]*agg{}
	byUC := map[string]*agg{}
	byErr := map[string]int32{}
	durs := make([]int64, 0, len(rows))
	var sum int64
	for i := range rows {
		row := rows[i]
		out.ToplamCagri++
		if row.Success {
			out.BasariliCagri++
		} else {
			out.BasarisizCagri++
			tip := row.ErrorType
			if tip == "" {
				tip = llm.HataYok
			}
			if tip != llm.HataYok {
				byErr[tip]++
			}
		}
		durs = append(durs, row.DurationMs)
		sum += row.DurationMs
		agent := row.Agent
		if agent == "" {
			agent = llm.AgentReader
		}
		if byAgent[agent] == nil {
			byAgent[agent] = &agg{}
		}
		byAgent[agent].n++
		byAgent[agent].sum += row.DurationMs
		if row.Success {
			byAgent[agent].ok++
		}
		uc := row.Endpoint
		if uc == "" {
			uc = llm.EndpointUC1
		}
		if byUC[uc] == nil {
			byUC[uc] = &agg{}
		}
		byUC[uc].n++
		byUC[uc].sum += row.DurationMs
		if row.Success {
			byUC[uc].ok++
		}
	}
	n := float64(len(rows))
	out.OrtalamaSureMs = float64(sum) / n
	out.P95SureMs = p95(durs)
	for _, name := range sortedKeys(byAgent) {
		a := byAgent[name]
		tip := model.PromptTipi(name)
		if tip != model.PromptTipiOkuyucu && tip != model.PromptTipiDenetci {
			tip = model.PromptTipiOkuyucu
		}
		out.AgentBazinda = append(out.AgentBazinda, &model.LlmAgentMetrik{
			Agent:          tip,
			Cagri:          int32(a.n),
			OrtalamaSureMs: float64(a.sum) / float64(a.n),
			BasariOrani:    float64(a.ok) / float64(a.n),
		})
	}
	for _, name := range sortedKeys(byUC) {
		a := byUC[name]
		out.UcBazinda = append(out.UcBazinda, &model.LlmUcMetrik{
			UcAdi:          name,
			Cagri:          int32(a.n),
			OrtalamaSureMs: float64(a.sum) / float64(a.n),
			BasariOrani:    float64(a.ok) / float64(a.n),
		})
	}
	for _, name := range []string{llm.HataTimeout, llm.HataHTTP5xx, llm.HataHTTP4xx, llm.HataParse} {
		if byErr[name] == 0 {
			continue
		}
		out.HataDagilimi = append(out.HataDagilimi, &model.LlmHataAdet{
			HataTipi: name,
			Adet:     byErr[name],
		})
	}
	return out
}

func sortedKeys[T any](m map[string]*T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func llmCallToModel(row *repository.LLMCall) *model.LlmCagri {
	agent := model.PromptTipiOkuyucu
	if row.Agent == llm.AgentAuditor {
		agent = model.PromptTipiDenetci
	}
	tipi := row.ErrorType
	if tipi == "" {
		tipi = llm.HataYok
	}
	return &model.LlmCagri{
		Agent:     agent,
		UcAdi:     row.Endpoint,
		SureMs:    int32(row.DurationMs),
		Basarili:  row.Success,
		HataTipi:  tipi,
		DenemeNo:  row.Attempt,
		Baslangic: row.Start,
	}
}

// p95 nearest-rank: 1-tabanlı sıra ceil(0.95*n). (n-1)*0.95 kırpımı
// küçük n'de düşük değer seçer; n<20 için ceil max'a düşer.
func p95(vals []int64) float64 {
	if len(vals) == 0 {
		return 0
	}
	cp := append([]int64(nil), vals...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	return float64(cp[p95Index(len(cp))])
}

func p95Index(n int) int {
	if n <= 0 {
		return 0
	}
	rank := int(math.Ceil(0.95 * float64(n)))
	if rank < 1 {
		rank = 1
	}
	if rank > n {
		rank = n
	}
	return rank - 1
}
