package auth

import (
	"strings"
	"testing"
)

func TestHashPasswordVeVerify(t *testing.T) {
	p := Params{Time: 1, Memory: 8 * 1024, Threads: 1, KeyLen: 32, SaltLen: 16}
	const sifre = "oniki-karakter"
	hash, err := HashPassword(sifre, p)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if strings.Contains(hash, sifre) {
		t.Fatal("özet düz şifreyi içeriyor")
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("PHC biçimi değil: %s", hash)
	}
	if err := VerifyPassword(sifre, hash); err != nil {
		t.Fatalf("doğrulama: %v", err)
	}
	if err := VerifyPassword("yanlis-sifredir", hash); err == nil {
		t.Fatal("yanlış şifre kabul edildi")
	}
}

func TestHashPasswordKisa(t *testing.T) {
	_, err := HashPassword("kisa", DefaultParams())
	if err != ErrPasswordTooShort {
		t.Fatalf("err = %v", err)
	}
}

func TestNormalizeEposta(t *testing.T) {
	got, err := NormalizeEposta("  Ali@Ornek.COM ")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if got != "ali@ornek.com" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeEposta("gecersiz"); err != ErrInvalidEmail {
		t.Fatalf("err = %v", err)
	}
}
