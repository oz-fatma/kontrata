package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSYSTEM_PROMPTHasExampleJSON(t *testing.T) {
	if !strings.Contains(SYSTEM_PROMPT, "SADECE JSON") {
		t.Fatal("örnek çıktılı kısa prompt bekleniyordu")
	}
	if !strings.Contains(SYSTEM_PROMPT, `"oda_kontenjanlari"`) {
		t.Fatal("örnek JSON yok")
	}
	if !strings.Contains(SYSTEM_PROMPT, `"meta"`) || !strings.Contains(SYSTEM_PROMPT, `"otel_adi"`) {
		t.Fatal("örnek çıktıda meta yok")
	}
	if strings.Contains(SYSTEM_PROMPT, "Çekirdek alanlar") {
		t.Fatal("eğitimdeki uzun şema özeti kullanılmamalı")
	}
	if !strings.Contains(SYSTEM_PROMPT, "tamamen_garantili|kismen_garantili") {
		t.Fatal("sozlesme_tipi enum listesi yok")
	}
	if !strings.Contains(SYSTEM_PROMPT, "sezon (yaz|kis|yillik|belirtilmemis)") {
		t.Fatal("sezon enum listesi yok")
	}
	if !strings.Contains(SYSTEM_PROMPT, "yetkili_mahkeme (sadece şehir adı)") {
		t.Fatal("yetkili_mahkeme kısıtı yok")
	}
	if !strings.Contains(SYSTEM_PROMPT, "meta alanı yalnızca en üstte bir kez yazılır") {
		t.Fatal("meta tek kök uyarısı yok")
	}
}

func TestSYSTEM_PROMPTMatchesMLCopies(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller yok")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	evalSrc, err := os.ReadFile(filepath.Join(root, "ml", "evaluate.py"))
	if err != nil {
		t.Fatalf("evaluate.py: %v", err)
	}
	evalPrompt, err := extractTripleQuotedPrompt(string(evalSrc))
	if err != nil {
		t.Fatalf("evaluate.py prompt: %v", err)
	}
	nbPrompt, err := notebookSystemPrompt(filepath.Join(root, "ml", "train_colab.ipynb"))
	if err != nil {
		t.Fatalf("train_colab.ipynb prompt: %v", err)
	}
	if evalPrompt != SYSTEM_PROMPT {
		t.Fatal("evaluate.py SYSTEM_PROMPT agent ile aynı değil")
	}
	if nbPrompt != SYSTEM_PROMPT {
		t.Fatal("train_colab.ipynb SYSTEM_PROMPT agent ile aynı değil")
	}
}

func notebookSystemPrompt(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var nb struct {
		Cells []struct {
			Source []string `json:"source"`
		} `json:"cells"`
	}
	if err := json.Unmarshal(raw, &nb); err != nil {
		return "", err
	}
	var b strings.Builder
	for _, c := range nb.Cells {
		for _, line := range c.Source {
			b.WriteString(line)
		}
	}
	return extractTripleQuotedPrompt(b.String())
}

func extractTripleQuotedPrompt(src string) (string, error) {
	const marker = `SYSTEM_PROMPT = """`
	i := strings.Index(src, marker)
	if i < 0 {
		return "", os.ErrNotExist
	}
	rest := src[i+len(marker):]
	j := strings.Index(rest, `"""`)
	if j < 0 {
		return "", os.ErrNotExist
	}
	return rest[:j], nil
}
