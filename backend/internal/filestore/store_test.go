package filestore

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStore_SaveOpenRemove(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.Save(bytes.NewReader([]byte("%PDF-1.4 dummy")))
	if err != nil {
		t.Fatal(err)
	}
	if !validID(id) {
		t.Fatalf("kimlik %q", id)
	}
	f, err := s.Open(id)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if err := s.Remove(id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, id)); !os.IsNotExist(err) {
		t.Fatal("dosya silinmedi")
	}
}

func TestStore_RejectsNonPDF(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Save(bytes.NewReader([]byte("not a pdf")))
	if err != ErrNotPDF {
		t.Fatalf("hata = %v", err)
	}
}

func TestStore_RejectsPathTraversal(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Open("../etc/passwd")
	if err != ErrInvalidID {
		t.Fatalf("hata = %v", err)
	}
	if err := s.Remove(".." + strings.Repeat("a", 30)); err != ErrInvalidID {
		t.Fatalf("hata = %v", err)
	}
}
