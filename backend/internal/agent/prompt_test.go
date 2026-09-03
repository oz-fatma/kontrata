package agent

import (
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
	if strings.Contains(SYSTEM_PROMPT, "Çekirdek alanlar") {
		t.Fatal("eğitimdeki uzun şema özeti kullanılmamalı")
	}
}
