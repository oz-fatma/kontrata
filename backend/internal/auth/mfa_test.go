package auth

import (
	"strings"
	"testing"
)

func TestNewMFACodeFormatAndHash(t *testing.T) {
	plain, hash, err := NewMFACode()
	if err != nil {
		t.Fatalf("NewMFACode: %v", err)
	}
	if len(plain) != MFADigits {
		t.Fatalf("uzunluk = %d", len(plain))
	}
	for _, r := range plain {
		if r < '0' || r > '9' {
			t.Fatalf("sayısal değil: %q", plain)
		}
	}
	if hash == plain {
		t.Fatal("hash düz kod")
	}
	if !MFACodeMatch(plain, hash) {
		t.Fatal("eşleşme başarısız")
	}
	if MFACodeMatch("000000", hash) && plain != "000000" {
		t.Fatal("yanlış kod eşleşti")
	}
}

func TestMFACodeLeadingZeros(t *testing.T) {
	if got := len("000123"); got != 6 {
		t.Fatal("sabit")
	}
	hash := HashToken("000123")
	if !MFACodeMatch("000123", hash) {
		t.Fatal("baştaki sıfırlar kayboldu")
	}
	if strings.Contains(hash, "000123") {
		t.Fatal("düz kod hash içinde")
	}
}
