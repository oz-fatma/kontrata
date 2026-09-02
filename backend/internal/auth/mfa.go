package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	// MFADigits kod uzunluğudur; baştaki sıfırlar korunur.
	MFADigits = 6
	// MFATTL kodun geçerlilik süresidir.
	MFATTL = 120 * time.Second
	// MFAMaxAttempts yanlış deneme üst sınırıdır.
	MFAMaxAttempts = 5
	// MFAPendingTTL geçici MFA jetonunun ömrüdür.
	MFAPendingTTL = 5 * time.Minute
)

var errMFA = fmt.Errorf("doğrulama kodu üretilemedi")

// NewMFACode 000000-999999 aralığında 6 haneli kod üretir; hash SHA-256'dır.
func NewMFACode() (plain, hash string, err error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", errMFA
	}
	n := binary.BigEndian.Uint32(b[:]) % 1_000_000
	plain = fmt.Sprintf("%06d", n)
	return plain, HashToken(plain), nil
}

// MFACodeMatch hash'leri sabit zamanda karşılaştırır. Düz kod loglanmaz.
func MFACodeMatch(plain, hash string) bool {
	got := HashToken(plain)
	return subtle.ConstantTimeCompare([]byte(got), []byte(hash)) == 1
}
