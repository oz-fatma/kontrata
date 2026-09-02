package pdf

import (
	"regexp"
	"strings"
)

const marginLines = 3

var extraBlankLines = regexp.MustCompile(`\n{3,}`)

// CleanPages tekrarlayan başlık/altbilgi satırlarını ayıklar ve boşlukları
// sadeleştirir. Satır içi çoklu boşluk (tablo hizası) ve madde numaraları korunur.
func CleanPages(pages []string) []string {
	normalized := make([]string, len(pages))
	for i, p := range pages {
		normalized[i] = normalizeLineEndings(p)
	}
	headers, footers := repeatedMarginLines(normalized)
	out := make([]string, len(normalized))
	for i, p := range normalized {
		out[i] = collapseBlankLines(stripMarginLines(p, headers, footers))
	}
	return out
}

func normalizeLineEndings(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

func collapseBlankLines(s string) string {
	s = extraBlankLines.ReplaceAllString(s, "\n\n")
	return strings.Trim(s, "\n")
}

func repeatedMarginLines(pages []string) (headers, footers map[string]struct{}) {
	if len(pages) < 2 {
		return nil, nil
	}
	hCount := make(map[string]int)
	fCount := make(map[string]int)
	for _, p := range pages {
		lines := strings.Split(p, "\n")
		for _, line := range takeNonEmpty(lines, marginLines, false) {
			hCount[line]++
		}
		for _, line := range takeNonEmpty(lines, marginLines, true) {
			fCount[line]++
		}
	}
	min := 2
	if len(pages) >= 4 {
		min = (len(pages) + 1) / 2
	}
	headers = pickRepeated(hCount, min)
	footers = pickRepeated(fCount, min)
	return headers, footers
}

func pickRepeated(counts map[string]int, min int) map[string]struct{} {
	out := make(map[string]struct{})
	for line, n := range counts {
		if n >= min && strings.TrimSpace(line) != "" {
			out[line] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func takeNonEmpty(lines []string, n int, fromEnd bool) []string {
	if n <= 0 {
		return nil
	}
	out := make([]string, 0, n)
	if !fromEnd {
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			out = append(out, line)
			if len(out) == n {
				break
			}
		}
		return out
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		out = append(out, lines[i])
		if len(out) == n {
			break
		}
	}
	return out
}

func stripMarginLines(page string, headers, footers map[string]struct{}) string {
	if len(headers) == 0 && len(footers) == 0 {
		return page
	}
	lines := strings.Split(page, "\n")
	drop := make([]bool, len(lines))
	seen := 0
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if seen < marginLines && inSet(headers, line) {
			drop[i] = true
		}
		seen++
		if seen >= marginLines {
			break
		}
	}
	seen = 0
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if seen < marginLines && inSet(footers, lines[i]) {
			drop[i] = true
		}
		seen++
		if seen >= marginLines {
			break
		}
	}
	kept := make([]string, 0, len(lines))
	for i, line := range lines {
		if drop[i] {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func inSet(set map[string]struct{}, line string) bool {
	if set == nil {
		return false
	}
	_, ok := set[line]
	return ok
}
