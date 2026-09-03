package agent

import (
	"context"
	"strings"
	"testing"
)

type recLLM struct {
	sys  []string
	user []string
	out  string
}

func (r *recLLM) Generate(_ context.Context, sys, user string) (string, error) {
	r.sys = append(r.sys, sys)
	r.user = append(r.user, user)
	if r.out == "" {
		return "{}", nil
	}
	return r.out, nil
}

func TestReader_UsesInjectedPrompt(t *testing.T) {
	rec := &recLLM{out: "{}"}
	r := &Reader{LLM: rec, SystemPrompt: "OZEL OKUYUCU PROMPT"}
	_, _ = r.Extract(context.Background(), []string{"kontenjan maddesi"})
	if len(rec.sys) == 0 || rec.sys[0] != "OZEL OKUYUCU PROMPT" {
		t.Fatalf("sistem prompt = %#v", rec.sys)
	}
}

func TestReader_MasksUserPrompt(t *testing.T) {
	rec := &recLLM{out: "{}"}
	r := &Reader{LLM: rec}
	_, _ = r.Extract(context.Background(), []string{
		"İletişim rezervasyon@otel.test tel 05321234567 kimlik 12345678901",
	})
	if len(rec.user) == 0 {
		t.Fatal("kullanıcı metni yok")
	}
	u := rec.user[0]
	for _, leak := range []string{"rezervasyon@otel.test", "05321234567", "12345678901"} {
		if strings.Contains(u, leak) {
			t.Fatalf("sızıntı %q: %s", leak, u)
		}
	}
	if !strings.Contains(u, "[EPOSTA]") || !strings.Contains(u, "[TELEFON]") || !strings.Contains(u, "[TCKN]") {
		t.Fatalf("jeton eksik: %s", u)
	}
}
