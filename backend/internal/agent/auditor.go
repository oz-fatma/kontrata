package agent

import (
	"context"
	"log"
	"sort"
	"time"

	"github.com/oz-fatma/kontrata/backend/internal/llm"
)

// Finding bir denetçi bulgusudur.
type Finding struct {
	Code        string
	Title       string
	Description string
	Severity    string
	Source      string
	FieldPath   string
}

// AuditResult kural ve model denetiminin birleşik çıktısıdır.
type AuditResult struct {
	Findings      []Finding
	RuleDuration  time.Duration
	ModelDuration time.Duration
}

// Auditor Okuyucu çıktısını kural motoru ve LLM ile denetler.
type Auditor struct {
	LLM llm.Client
}

// Audit önce kuralları, sonra yoruma dayalı LLM denetimini çalıştırır.
// LLM katmanı başarısız olsa da kural bulguları döner; hata üretilmez.
func (a *Auditor) Audit(ctx context.Context, data map[string]any, pages []string) (*AuditResult, error) {
	log.Printf("denetci basladi")
	start := time.Now()
	findings := runRules(data)
	ruleDur := time.Since(start)

	modelStart := time.Now()
	extra := a.llmFindings(ctx, data, pages)
	modelDur := time.Since(modelStart)

	all := append(findings, extra...)
	sortFindings(all)
	if all == nil {
		all = []Finding{}
	}
	log.Printf("denetci bitti kural=%d model=%d sure=%s", len(findings), len(extra), time.Since(start))
	return &AuditResult{
		Findings:      all,
		RuleDuration:  ruleDur,
		ModelDuration: modelDur,
	}, nil
}

func sortFindings(in []Finding) {
	rank := func(s string) int {
		switch s {
		case SeverityCritical:
			return 0
		case SeverityWarning:
			return 1
		case SeverityInfo:
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(in, func(i, j int) bool {
		ri, rj := rank(in[i].Severity), rank(in[j].Severity)
		if ri != rj {
			return ri < rj
		}
		if in[i].Source != in[j].Source {
			return in[i].Source == SourceRule
		}
		return in[i].Code < in[j].Code
	})
}
