package mailer

import "testing"

func TestMaskEmail(t *testing.T) {
	got := MaskEmail("fatma@ornek.com")
	if got != "f***@ornek.com" {
		t.Fatalf("got %q", got)
	}
	if MaskEmail("a@b.co") != "a***@b.co" {
		t.Fatalf("tek karakter: %q", MaskEmail("a@b.co"))
	}
	if MaskEmail("yok") != "***" {
		t.Fatalf("geçersiz: %q", MaskEmail("yok"))
	}
}
