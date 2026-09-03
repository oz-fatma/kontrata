package mask

import (
	"strings"
	"testing"
)

func TestApply_Email(t *testing.T) {
	in := "İletişim: rezervasyon@otel-ornek.com ve Ali Yılmaz"
	got := Apply(in)
	if got.Count != 1 {
		t.Fatalf("sayı = %d", got.Count)
	}
	if strings.Contains(got.Text, "rezervasyon@otel-ornek.com") {
		t.Fatalf("e-posta kaldı: %s", got.Text)
	}
	if !strings.Contains(got.Text, "[EPOSTA]") {
		t.Fatalf("jeton yok: %s", got.Text)
	}
}

func TestApply_Phone05xx(t *testing.T) {
	cases := []string{
		"Tel: 0532 123 45 67",
		"Tel: 05321234567",
		"Tel: 0532 123 45\n67",
	}
	for _, in := range cases {
		got := Apply(in)
		if got.Count < 1 {
			t.Fatalf("%q sayı = %d", in, got.Count)
		}
		if strings.Contains(got.Text, "532") && strings.Contains(got.Text, "123") {
			t.Fatalf("telefon kaldı (%q): %s", in, got.Text)
		}
		if !strings.Contains(got.Text, "[TELEFON]") {
			t.Fatalf("jeton yok (%q): %s", in, got.Text)
		}
	}
}

func TestApply_PhonePlus90(t *testing.T) {
	cases := []string{
		"Tel: +90 532 123 45 67",
		"Tel: +905321234567",
		"Tel: +90 242 813 45 67",
		"Tel: +90 242 813 45\n67",
		"Tel: +90 532 123 45\n67",
	}
	for _, in := range cases {
		got := Apply(in)
		if got.Count < 1 {
			t.Fatalf("%q sayı = %d", in, got.Count)
		}
		if strings.Contains(got.Text, "+90") {
			t.Fatalf("+90 kaldı (%q): %s", in, got.Text)
		}
		if !strings.Contains(got.Text, "[TELEFON]") {
			t.Fatalf("jeton yok (%q): %s", in, got.Text)
		}
	}
}

func TestApply_TCKN(t *testing.T) {
	in := "Kimlik no 12345678901 sözleşmede yazılıdır."
	got := Apply(in)
	if got.Count != 1 {
		t.Fatalf("sayı = %d metin=%s", got.Count, got.Text)
	}
	if strings.Contains(got.Text, "12345678901") {
		t.Fatalf("TCKN kaldı: %s", got.Text)
	}
	if !strings.Contains(got.Text, "[TCKN]") {
		t.Fatalf("jeton yok: %s", got.Text)
	}
}

func TestApply_DoesNotLogContent(t *testing.T) {
	in := "a@b.co 05321234567 12345678901"
	got := Apply(in)
	if got.Count != 3 {
		t.Fatalf("sayı = %d metin=%s", got.Count, got.Text)
	}
	for _, leak := range []string{"a@b.co", "05321234567", "12345678901"} {
		if strings.Contains(got.Text, leak) {
			t.Fatalf("sızıntı %q: %s", leak, got.Text)
		}
	}
}

func TestApply_FivePIIFromWrappedPDF(t *testing.T) {
	in := "kimlik no 12345678901. Rezervasyon bildirimleri\n" +
		"ayse.demir@argosotel.com adresine ve 0532 445 67 89 numaralı telefona yapılır.\n" +
		"İletişim: m.yildiz@sideturizm.com.tr, +90 242 813 45\n67.\n"
	got := Apply(in)
	if got.Count != 5 {
		t.Fatalf("sayı = %d metin=%s", got.Count, got.Text)
	}
	for _, leak := range []string{
		"12345678901",
		"ayse.demir@argosotel.com",
		"0532 445 67 89",
		"m.yildiz@sideturizm.com.tr",
		"+90 242 813 45",
		"813 45",
	} {
		if strings.Contains(got.Text, leak) {
			t.Fatalf("sızıntı %q: %s", leak, got.Text)
		}
	}
	if strings.Count(got.Text, "[EPOSTA]") != 2 {
		t.Fatalf("e-posta jetonu = %d metin=%s", strings.Count(got.Text, "[EPOSTA]"), got.Text)
	}
	if strings.Count(got.Text, "[TELEFON]") != 2 {
		t.Fatalf("telefon jetonu = %d metin=%s", strings.Count(got.Text, "[TELEFON]"), got.Text)
	}
	if strings.Count(got.Text, "[TCKN]") != 1 {
		t.Fatalf("TCKN jetonu yok: %s", got.Text)
	}
}

func TestApply_Empty(t *testing.T) {
	got := Apply("")
	if got.Count != 0 || got.Text != "" {
		t.Fatalf("%+v", got)
	}
}
