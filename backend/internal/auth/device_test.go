package auth

import (
	"strings"
	"testing"
)

func TestDeviceFingerprintStableAndHashed(t *testing.T) {
	a := DeviceFingerprint("abc", "UA/1", "tr-TR")
	b := DeviceFingerprint("abc", "UA/1", "tr-TR")
	if a != b || a == "" {
		t.Fatal("parmak izi kararlı değil")
	}
	if strings.Contains(a, "abc") || strings.Contains(a, "UA/1") {
		t.Fatal("ham parmak izi yazılmış")
	}
	if DeviceFingerprint("abc", "UA/1", "en") == a {
		t.Fatal("dil değişince iz aynı kaldı")
	}
}

func TestDeviceFingerprintFallsBackToUA(t *testing.T) {
	with := DeviceFingerprint("id-1", "Mozilla/5.0", "tr")
	without := DeviceFingerprint("", "Mozilla/5.0", "tr")
	if with == without {
		t.Fatal("cihaz kimliği yok sayıldı")
	}
	again := DeviceFingerprint("", "Mozilla/5.0", "tr")
	if without != again {
		t.Fatal("tarayıcı izi kararlı değil")
	}
}

func TestDeviceLabelChromeMac(t *testing.T) {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	if got := DeviceLabel(ua); got != "Chrome / macOS" {
		t.Fatalf("got %q", got)
	}
}
