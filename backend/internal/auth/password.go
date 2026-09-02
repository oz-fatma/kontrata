package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	// MinPasswordLen OWASP önerisine uygun asgari uzunluktur.
	MinPasswordLen = 12
	argon2Version  = argon2.Version
)

// ErrPasswordTooShort kullanıcıya dönebilen doğrulama hatasıdır.
var ErrPasswordTooShort = errors.New("şifre en az 12 karakter olmalı")

var (
	errHashFailed   = errors.New("şifre işlenemedi")
	errHashMismatch = errors.New("şifre doğrulanamadı")
)

// Params argon2id maliyetidir. Hash içinde saklanır; doğrulama kayıtlı değerleri kullanır.
type Params struct {
	Time    uint32
	Memory  uint32 // KiB
	Threads uint8
	KeyLen  uint32
	SaltLen uint32
}

// DefaultParams OWASP Password Storage Cheat Sheet (argon2id) önerisidir:
// m=19456 KiB (19 MiB), t=2, p=1.
func DefaultParams() Params {
	return Params{
		Time:    2,
		Memory:  19 * 1024,
		Threads: 1,
		KeyLen:  32,
		SaltLen: 16,
	}
}

// HashPassword argon2id ile PHC biçiminde özet üretir. Düz şifre asla loglanmaz.
func HashPassword(password string, p Params) (string, error) {
	if utf8.RuneCountInString(password) < MinPasswordLen {
		return "", ErrPasswordTooShort
	}
	if p.Time == 0 || p.Memory == 0 || p.Threads == 0 || p.KeyLen == 0 || p.SaltLen == 0 {
		p = DefaultParams()
	}
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", errHashFailed
	}
	sum := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, p.KeyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version, p.Memory, p.Time, p.Threads,
		b64.EncodeToString(salt), b64.EncodeToString(sum),
	), nil
}

// VerifyPassword saklı özetle karşılaştırır. Süre yan kanalını azaltmak için sabit zamanlıdır.
func VerifyPassword(password, encoded string) error {
	p, salt, want, err := parsePHC(encoded)
	if err != nil {
		return errHashMismatch
	}
	got := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, p.KeyLen)
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return errHashMismatch
	}
	return nil
}

func parsePHC(encoded string) (Params, []byte, []byte, error) {
	// $argon2id$v=19$m=19456,t=2,p=1$salt$hash
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Params{}, nil, nil, errHashMismatch
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2Version {
		return Params{}, nil, nil, errHashMismatch
	}
	var p Params
	for _, kv := range strings.Split(parts[3], ",") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return Params{}, nil, nil, errHashMismatch
		}
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return Params{}, nil, nil, errHashMismatch
		}
		switch k {
		case "m":
			p.Memory = uint32(n)
		case "t":
			p.Time = uint32(n)
		case "p":
			p.Threads = uint8(n)
		default:
			return Params{}, nil, nil, errHashMismatch
		}
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return Params{}, nil, nil, errHashMismatch
	}
	sum, err := b64.DecodeString(parts[5])
	if err != nil {
		return Params{}, nil, nil, errHashMismatch
	}
	p.SaltLen = uint32(len(salt))
	p.KeyLen = uint32(len(sum))
	if p.Time == 0 || p.Memory == 0 || p.Threads == 0 || p.KeyLen == 0 {
		return Params{}, nil, nil, errHashMismatch
	}
	return p, salt, sum, nil
}
