package auth

import (
	"strings"
	"testing"
)

func TestHashPasswordAndVerify(t *testing.T) {
	p := Params{Time: 1, Memory: 8 * 1024, Threads: 1, KeyLen: 32, SaltLen: 16}
	const password = "oniki-karakter"
	hash, err := HashPassword(password, p)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if strings.Contains(hash, password) {
		t.Fatal("özet düz şifreyi içeriyor")
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("PHC biçimi değil: %s", hash)
	}
	if err := VerifyPassword(password, hash); err != nil {
		t.Fatalf("doğrulama: %v", err)
	}
	if err := VerifyPassword("yanlis-sifredir", hash); err == nil {
		t.Fatal("yanlış şifre kabul edildi")
	}
}

func TestHashPasswordTooShort(t *testing.T) {
	_, err := HashPassword("kisa", DefaultParams())
	if err != ErrPasswordTooShort {
		t.Fatalf("err = %v", err)
	}
}

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail("  Ali@Ornek.COM ")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if got != "ali@ornek.com" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeEmail("gecersiz"); err != ErrInvalidEmail {
		t.Fatalf("err = %v", err)
	}
}
