package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// AccessTTL erişim jetonu ömrüdür.
	AccessTTL = 15 * time.Minute
	// RefreshTTL yenileme jetonu ömrüdür.
	RefreshTTL = 7 * 24 * time.Hour

	purposeMFA    = "mfa"
	purposeAccess = "access"
)

var (
	// ErrUnauthorized GraphQL'e dönen kimlik hatasıdır; jeton ayrıntısı yok.
	ErrUnauthorized = errors.New("kimlik doğrulaması gerekli")
	// ErrGecersizYenilemeJetonu yenileme jetonu kabul edilmediğinde döner.
	// Süre, iptal ve yokluk ayırt edilmez.
	ErrGecersizYenilemeJetonu = errors.New("oturum sonlandı, tekrar giriş yapın")
	// ErrMFAFailed MFA adımı için genel hatadır.
	ErrMFAFailed = errors.New("doğrulama başarısız")
)

var errJWT = errors.New("jeton geçersiz")

// JWT HS256 imzalayıcı ve doğrulayıcıdır.
type JWT struct {
	secret []byte
}

// NewJWT gizli anahtar boşsa hata döner.
func NewJWT(secret []byte) (*JWT, error) {
	if len(secret) == 0 {
		return nil, errors.New("JWT_SECRET eksik")
	}
	return &JWT{secret: secret}, nil
}

type accessClaims struct {
	KullaniciID string `json:"kullaniciId"`
	OturumID    string `json:"oturumId"`
	SonKullanma int64  `json:"sonKullanma"`
	Amac        string `json:"amac"`
	jwt.RegisteredClaims
}

type pendingClaims struct {
	KullaniciID string `json:"kullaniciId"`
	Amac        string `json:"amac"`
	jwt.RegisteredClaims
}

// SignAccess kullaniciId ve oturumId içerir; e-posta koyulmaz.
func (j *JWT) SignAccess(kullaniciID, oturumID string, now time.Time) (string, error) {
	if j == nil {
		return "", errJWT
	}
	exp := now.Add(AccessTTL)
	claims := accessClaims{
		KullaniciID: kullaniciID,
		OturumID:    oturumID,
		SonKullanma: exp.Unix(),
		Amac:        purposeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(j.secret)
	if err != nil {
		return "", errJWT
	}
	return s, nil
}

// ParseAccess erişim jetonunu çözer. Hata ayrıntısı sızmaz.
func (j *JWT) ParseAccess(token string) (kullaniciID, oturumID string, err error) {
	var claims accessClaims
	if err := j.parse(token, &claims); err != nil {
		return "", "", ErrUnauthorized
	}
	if claims.Amac != purposeAccess || claims.KullaniciID == "" || claims.OturumID == "" {
		return "", "", ErrUnauthorized
	}
	return claims.KullaniciID, claims.OturumID, nil
}

// SignPending MFA adımını bağlayan kısa ömürlü jetondur.
func (j *JWT) SignPending(kullaniciID string, now time.Time) (string, error) {
	if j == nil {
		return "", errJWT
	}
	claims := pendingClaims{
		KullaniciID: kullaniciID,
		Amac:        purposeMFA,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(MFAPendingTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(j.secret)
	if err != nil {
		return "", errJWT
	}
	return s, nil
}

// ParsePending geçici MFA jetonunu çözer.
func (j *JWT) ParsePending(token string) (kullaniciID string, err error) {
	var claims pendingClaims
	if err := j.parse(token, &claims); err != nil {
		return "", ErrMFAFailed
	}
	if claims.Amac != purposeMFA || claims.KullaniciID == "" {
		return "", ErrMFAFailed
	}
	return claims.KullaniciID, nil
}

func (j *JWT) parse(token string, claims jwt.Claims) error {
	if j == nil || token == "" {
		return errJWT
	}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("alg")
		}
		return j.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || parsed == nil || !parsed.Valid {
		return errJWT
	}
	return nil
}
