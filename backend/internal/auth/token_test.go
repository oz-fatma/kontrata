package auth

import (
	"encoding/base64"
	"testing"
)

func TestNewTokenStoresHash(t *testing.T) {
	plain, hash, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if plain == "" || hash == "" {
		t.Fatal("boş token")
	}
	if plain == hash {
		t.Fatal("hash düz metinle aynı")
	}
	if HashToken(plain) != hash {
		t.Fatal("HashToken tutarsız")
	}
	raw, err := base64.RawURLEncoding.DecodeString(plain)
	if err != nil {
		t.Fatalf("base64: %v", err)
	}
	if len(raw) != TokenBytes {
		t.Fatalf("uzunluk = %d", len(raw))
	}
}
