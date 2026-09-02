// Package pdf kontenjan sözleşmesi PDF'lerinden sayfa sayfa düz metin çıkarır.
// Model çağrısı yoktur; OCR yoktur.
package pdf

import (
	"bytes"
	"errors"
	"io"
	"log"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	pdfread "github.com/ledongthuc/pdf"
)

const lineYTolerance = 3.0

// ErrNoTextLayer taranmış veya yalnızca görüntü içeren PDF'ler içindir.
// OCR bilinçli olarak desteklenmez.
var ErrNoTextLayer = errors.New("PDF'de metin katmanı yok")

// ErrEmpty sayfa içermeyen PDF içindir.
var ErrEmpty = errors.New("PDF boş")

// ErrUnreadable dosyanın PDF olarak açılamadığını belirtir.
// Ayrıntı loga yazılır; sözleşme metni loglanmaz.
var ErrUnreadable = errors.New("PDF okunamadı")

// DocumentInfo çıkarılmış metnin özetidir. Sözleşme gövdesi içermez.
type DocumentInfo struct {
	PageCount int
	CharCount int
}

// ExtractText her sayfanın düz metnini ayrı döner. Kaynak sayfa numarası
// (1 tabanlı) dilim indeksidir. Taranmış PDF'lerde ErrNoTextLayer döner.
func ExtractText(r io.Reader) ([]string, error) {
	raw, err := readPages(r)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		log.Printf("pdf boş sayfa=0 karakter=0")
		return nil, ErrEmpty
	}
	if !hasText(raw) {
		log.Printf("pdf metin katmanı yok sayfa=%d karakter=0", len(raw))
		return nil, ErrNoTextLayer
	}
	pages := CleanPages(raw)
	info := Info(pages)
	log.Printf("pdf metin çıkarıldı sayfa=%d karakter=%d", info.PageCount, info.CharCount)
	return pages, nil
}

// Info çıkarılmış sayfalardan sayfa ve karakter sayısı döner.
func Info(pages []string) DocumentInfo {
	n := 0
	for _, p := range pages {
		n += utf8.RuneCountInString(p)
	}
	return DocumentInfo{PageCount: len(pages), CharCount: n}
}

func readPages(r io.Reader) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		log.Printf("pdf okunamadı: %v", err)
		return nil, ErrUnreadable
	}
	reader, err := pdfread.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		log.Printf("pdf ayrıştırılamadı: %v", err)
		return nil, ErrUnreadable
	}
	n := reader.NumPage()
	pages := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		text, err := pageText(reader.Page(i))
		if err != nil {
			log.Printf("pdf sayfa metni alınamadı sayfa=%d: %v", i, err)
			return nil, ErrUnreadable
		}
		pages = append(pages, text)
	}
	return pages, nil
}

func pageText(p pdfread.Page) (text string, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = errors.New("sayfa yorumlanamadı")
		}
	}()
	if p.V.IsNull() || p.V.Key("Contents").Kind() == pdfread.Null {
		return "", nil
	}
	return layoutText(p.Content().Text), nil
}

func layoutText(items []pdfread.Text) string {
	filtered := make([]pdfread.Text, 0, len(items))
	for _, t := range items {
		if t.S == "" {
			continue
		}
		filtered = append(filtered, t)
	}
	if len(filtered) == 0 {
		return ""
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if math.Abs(filtered[i].Y-filtered[j].Y) > lineYTolerance {
			return filtered[i].Y > filtered[j].Y
		}
		return filtered[i].X < filtered[j].X
	})

	var lines [][]pdfread.Text
	var cur []pdfread.Text
	var curY float64
	for i, t := range filtered {
		if i == 0 || math.Abs(t.Y-curY) <= lineYTolerance {
			if i == 0 {
				curY = t.Y
			}
			cur = append(cur, t)
			continue
		}
		lines = append(lines, cur)
		cur = []pdfread.Text{t}
		curY = t.Y
	}
	if len(cur) > 0 {
		lines = append(lines, cur)
	}

	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(joinLine(line))
	}
	return b.String()
}

func joinLine(items []pdfread.Text) string {
	var b strings.Builder
	lastEnd := 0.0
	lastSpace := true
	for i, t := range items {
		if i > 0 && t.S != " " && !lastSpace {
			n := gapSpaces(t.X-lastEnd, t.FontSize)
			if n > 0 {
				b.WriteString(strings.Repeat(" ", n))
			}
		}
		b.WriteString(t.S)
		lastSpace = t.S == " " || strings.HasSuffix(t.S, " ")
		w := t.W
		if w <= 0 && t.FontSize > 0 {
			w = t.FontSize * 0.5 * float64(utf8.RuneCountInString(t.S))
		}
		lastEnd = t.X + w
	}
	return b.String()
}

func gapSpaces(gap, fontSize float64) int {
	if fontSize <= 0 {
		fontSize = 12
	}
	if gap < fontSize*0.25 {
		return 0
	}
	n := int(math.Round(gap / (fontSize * 0.5)))
	if n < 1 {
		n = 1
	}
	if n > 24 {
		n = 24
	}
	return n
}

func hasText(pages []string) bool {
	for _, p := range pages {
		if strings.TrimSpace(p) != "" {
			return true
		}
	}
	return false
}
