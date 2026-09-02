package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleContractPDF() []byte {
	return encodePDF([][]string{
		{
			"KONTENJAN SOZLESMESI",
			"MADDE 1 - SURE",
			"1 Ocak 2026 - 31 Aralik 2026",
			"standart    170",
			"Gizli",
		},
		{
			"KONTENJAN SOZLESMESI",
			"MADDE 2 - KONTENJAN",
			"suit    20",
			"Gizli",
		},
	})
}

func TestExtractText_PagesAndContent(t *testing.T) {
	src := sampleContractPDF()
	pages, err := ExtractText(bytes.NewReader(src))
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("sayfa sayısı = %d, beklenen 2", len(pages))
	}
	if !strings.Contains(pages[0], "MADDE 1") {
		t.Fatalf("1. sayfada madde numarası yok")
	}
	if !strings.Contains(pages[1], "MADDE 2") {
		t.Fatalf("2. sayfada madde numarası yok")
	}
	if strings.Contains(pages[0], "KONTENJAN SOZLESMESI") || strings.Contains(pages[1], "KONTENJAN SOZLESMESI") {
		t.Fatalf("tekrarlayan baslik ayiklanmadi")
	}
	if strings.Contains(pages[0], "Gizli") || strings.Contains(pages[1], "Gizli") {
		t.Fatalf("tekrarlayan altbilgi ayiklanmadi")
	}
	if !strings.Contains(pages[0], "standart    170") {
		t.Fatalf("tablo hizasi bozuldu: %q", pages[0])
	}
	info := Info(pages)
	if info.PageCount != 2 {
		t.Fatalf("Info.PageCount = %d", info.PageCount)
	}
	if info.CharCount == 0 {
		t.Fatalf("Info.CharCount = 0")
	}
}

func TestExtractText_TestdataFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "sample.pdf"))
	if err != nil {
		t.Fatalf("testdata/sample.pdf: %v", err)
	}
	pages, err := ExtractText(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("sayfa sayısı = %d, beklenen 2", len(pages))
	}
	joined := pages[0] + "\n" + pages[1]
	if !strings.Contains(joined, "MADDE 1") || !strings.Contains(joined, "MADDE 2") {
		t.Fatalf("testdata orneginden madde metni cikmadi")
	}
}

func TestExtractText_EmptyPDF(t *testing.T) {
	_, err := ExtractText(bytes.NewReader(emptyPDF()))
	if !errors.Is(err, ErrEmpty) {
		t.Fatalf("err = %v, beklenen ErrEmpty", err)
	}
}

func TestExtractText_NoTextLayer(t *testing.T) {
	_, err := ExtractText(bytes.NewReader(noTextPDF()))
	if !errors.Is(err, ErrNoTextLayer) {
		t.Fatalf("err = %v, beklenen ErrNoTextLayer", err)
	}
}

func TestExtractText_Unreadable(t *testing.T) {
	_, err := ExtractText(bytes.NewReader([]byte("bu bir pdf degil")))
	if !errors.Is(err, ErrUnreadable) {
		t.Fatalf("err = %v, beklenen ErrUnreadable", err)
	}
}

func TestExtractText_DoesNotLogContent(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	secret := "GIZLI-KONTENJAN-12345"
	src := encodePDF([][]string{
		{"MADDE 1", secret, "standart    10"},
	})
	pages, err := ExtractText(bytes.NewReader(src))
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	if !strings.Contains(pages[0], secret) {
		t.Fatalf("cikarilan metinde beklenen icerik yok")
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("sozlesme icerigi loglandi")
	}
	if !strings.Contains(buf.String(), "sayfa=1") {
		t.Fatalf("sayfa meta bilgisi loglanmadi: %s", buf.String())
	}
}

func TestCleanPages_NormalizeBlankLines(t *testing.T) {
	got := CleanPages([]string{"MADDE 1\n\n\n\nMADDE 2"})
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if strings.Contains(got[0], "\n\n\n") {
		t.Fatalf("fazla bos satir kaldi: %q", got[0])
	}
	if !strings.Contains(got[0], "MADDE 1") || !strings.Contains(got[0], "MADDE 2") {
		t.Fatalf("madde numarasi kayboldu: %q", got[0])
	}
}

func TestCleanPages_SinglePageKeepsHeader(t *testing.T) {
	got := CleanPages([]string{"KONTENJAN SOZLESMESI\nMADDE 1"})
	if !strings.Contains(got[0], "KONTENJAN SOZLESMESI") {
		t.Fatalf("tek sayfada baslik silinmemeli: %q", got[0])
	}
}

func encodePDF(pages [][]string) []byte {
	n := len(pages)
	objs := make([]string, 1, 3+2*n)
	kids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		kids = append(kids, fmt.Sprintf("%d 0 R", 4+2*i))
	}
	objs = append(objs, "<< /Type /Catalog /Pages 2 0 R >>")
	objs = append(objs, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), n))
	objs = append(objs, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	for i, lines := range pages {
		pageNo := 4 + 2*i
		contentNo := pageNo + 1
		objs = append(objs, fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents %d 0 R /Resources << /Font << /F1 3 0 R >> >> >>",
			contentNo,
		))
		objs = append(objs, contentStream(lines))
	}
	return assemblePDF(objs)
}

func emptyPDF() []byte {
	return assemblePDF([]string{
		"",
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [] /Count 0 >>",
	})
}

func noTextPDF() []byte {
	body := "q\n0 0 612 792 re\nf\nQ\n"
	content := fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(body), body)
	return assemblePDF([]string{
		"",
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << >> >>",
		content,
	})
}

func contentStream(lines []string) string {
	var inner strings.Builder
	inner.WriteString("BT\n/F1 12 Tf\n72 720 Td\n")
	for i, line := range lines {
		if i > 0 {
			inner.WriteString("0 -18 Td\n")
		}
		inner.WriteString("(")
		inner.WriteString(pdfEscape(line))
		inner.WriteString(") Tj\n")
	}
	inner.WriteString("ET\n")
	body := inner.String()
	return fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(body), body)
}

func pdfEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `(`, `\(`)
	s = strings.ReplaceAll(s, `)`, `\)`)
	return s
}

func assemblePDF(objs []string) []byte {
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i := 1; i < len(objs); i++ {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i, objs[i])
	}
	startxref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n", len(objs))
	b.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(objs); i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs), startxref)
	return []byte(b.String())
}
