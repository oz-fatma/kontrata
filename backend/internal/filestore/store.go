// Package filestore sözleşme PDF'lerini yerel diskte tutar.
// Bulut depolama yoktur; dosya adı UUID'dir, orijinal ad veritabanındadır.
package filestore

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxBytes    = 32 << 20
	idBytes     = 16
	pdfMagic    = "%PDF"
	defaultPerm = 0o700
)

var (
	// ErrNotPDF PDF sihirli baytı taşımayan yüklemeler içindir.
	ErrNotPDF = errors.New("yalnızca PDF yüklenir")
	// ErrTooLarge 32 MB üstü dosyalar içindir.
	ErrTooLarge = errors.New("dosya çok büyük")
	// ErrInvalidID yol kaçışı veya bozuk kimlik içindir.
	ErrInvalidID = errors.New("geçersiz dosya kimliği")
	// ErrNotFound silinecek veya açılacak dosya yoksa döner.
	ErrNotFound = errors.New("dosya bulunamadı")
)

// Store yerel bir dizin üzerinde UUID adlı dosyalar tutar.
type Store struct {
	dir string
}

// New dizini oluşturur (yoksa) ve deposu döner.
func New(dir string) (*Store, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = filepath.Join("data", "uploads")
	}
	if err := os.MkdirAll(dir, defaultPerm); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// Save içeriği UUID adıyla yazar. Kimliği döner.
func (s *Store) Save(r io.Reader) (string, error) {
	if s == nil {
		return "", ErrNotFound
	}
	limited := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if len(data) > maxBytes {
		return "", ErrTooLarge
	}
	if len(data) < 5 || !strings.HasPrefix(string(data[:5]), pdfMagic) {
		return "", ErrNotPDF
	}
	id, err := newID()
	if err != nil {
		return "", err
	}
	path, err := s.path(id)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return id, nil
}

// Open kimliğe karşılık gelen dosyayı okumak için açar.
func (s *Store) Open(id string) (*os.File, error) {
	path, err := s.path(id)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

// Remove kimliğe karşılık gelen dosyayı siler. Yoksa sessizce döner.
func (s *Store) Remove(id string) error {
	if id == "" {
		return nil
	}
	path, err := s.path(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// RemoveAll verilen kimlikleri siler. İlk gerçek hata döner.
func (s *Store) RemoveAll(ids []string) error {
	var first error
	for _, id := range ids {
		if err := s.Remove(id); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (s *Store) path(id string) (string, error) {
	if !validID(id) {
		return "", ErrInvalidID
	}
	return filepath.Join(s.dir, id), nil
}

func validID(id string) bool {
	if len(id) != idBytes*2 {
		return false
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func newID() (string, error) {
	b := make([]byte, idBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
