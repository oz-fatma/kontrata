package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"
)

const (
	// TokenBytes rastgele ham token uzunluğudur.
	TokenBytes = 32
	// TokenTTL e-posta doğrulama kodunun geçerlilik süresidir.
	TokenTTL = 24 * time.Hour
	// PasswordResetTTL şifre sıfırlama kodunun geçerlilik süresidir (doğrulamadan kısa).
	PasswordResetTTL = 1 * time.Hour
	// AccountDeleteTTL hesap silme onay kodunun geçerlilik süresidir.
	AccountDeleteTTL = 1 * time.Hour
)

var errTokenFailed = errors.New("doğrulama kodu üretilemedi")

// NewToken 32 bayt rastgele değer üretir. Dönen düz metin yalnızca e-postaya gider;
// hash veritabanına yazılır.
func NewToken() (plaintext, hash string, err error) {
	raw := make([]byte, TokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", errTokenFailed
	}
	plaintext = base64.RawURLEncoding.EncodeToString(raw)
	return plaintext, HashToken(plaintext), nil
}

// HashToken SHA-256 özetini hex olarak döner. Düz metin saklanmaz.
func HashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
