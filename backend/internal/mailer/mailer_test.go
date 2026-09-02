package mailer

import "testing"

func TestMaskEposta(t *testing.T) {
	got := MaskEposta("fatma@ornek.com")
	if got != "f***@ornek.com" {
		t.Fatalf("got %q", got)
	}
	if MaskEposta("a@b.co") != "a***@b.co" {
		t.Fatalf("tek karakter: %q", MaskEposta("a@b.co"))
	}
	if MaskEposta("yok") != "***" {
		t.Fatalf("geçersiz: %q", MaskEposta("yok"))
	}
}
