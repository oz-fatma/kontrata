package service

import (
	"errors"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/filestore"
)

// ServeFile GET /dosya/{id} — yüklenen PDF'i yetkili kullanıcıya verir.
// Başka kiracının kaydı ve eksik dosya 404 döner; varlık sızmaz.
func (s *ContractService) ServeFile(w http.ResponseWriter, r *http.Request) {
	if s == nil {
		http.NotFound(w, r)
		return
	}
	if _, ok := auth.IdentityFrom(r.Context()); !ok {
		http.Error(w, auth.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}
	id := fileRequestID(r)
	if id == "" {
		http.NotFound(w, r)
		return
	}
	act, err := s.actor(r.Context())
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			http.Error(w, auth.ErrUnauthorized.Error(), http.StatusUnauthorized)
			return
		}
		log.Printf("dosya yetki okunamadı: %v", err)
		http.Error(w, "işlem tamamlanamadı", http.StatusInternalServerError)
		return
	}
	if !act.can(opContractRead) {
		http.NotFound(w, r)
		return
	}
	doc, err := s.repo.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !act.ownsContract(doc) || doc.StoredFileID == "" || s.files == nil {
		http.NotFound(w, r)
		return
	}
	f, err := s.files.Open(doc.StoredFileID)
	if err != nil {
		if !errors.Is(err, filestore.ErrNotFound) && !errors.Is(err, filestore.ErrInvalidID) {
			log.Printf("dosya açılamadı: %v", err)
		}
		http.NotFound(w, r)
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Printf("dosya kapatılamadı: %v", cerr)
		}
	}()
	name := "sozlesme.pdf"
	if doc.FileName != nil {
		base := filepath.Base(strings.TrimSpace(*doc.FileName))
		if base != "" && base != "." && base != string(filepath.Separator) {
			name = base
		}
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+strings.ReplaceAll(name, `"`, "")+`"`)
	w.Header().Set("Cache-Control", "private, no-store")
	if _, err := io.Copy(w, f); err != nil {
		log.Printf("dosya yazılamadı: %v", err)
	}
}

func fileRequestID(r *http.Request) string {
	if id := chi.URLParam(r, "id"); id != "" {
		return id
	}
	return r.PathValue("id")
}
