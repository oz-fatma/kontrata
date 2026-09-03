// Package extract ham model çıktısını onarır, şemaya çeker ve doğrular.
// Model çağrısı yoktur; girdi düz metindir.
package extract

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"
)

// DebugDump true ise onarılamayan çıktının kısa kesiti loglanır.
// Yalnızca LLM_DEBUG_DUMP ile açılır; üretimde kapalı kalmalıdır.
var DebugDump bool

// ErrUnparseable JSON nesnesi çıkarılamadığında döner.
var ErrUnparseable = errors.New("model çıktısı JSON olarak çözülemedi")

// RepairJSON ham LLM metninden bir JSON nesnesi üretir.
func RepairJSON(raw string) (map[string]any, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		log.Printf("json ayrıştırılamadı neden=bos")
		return nil, ErrUnparseable
	}
	s = stripMarkdown(s)

	span := objectSpan(s)
	if obj, ok := decodeObject(span); ok {
		return finishRepair(obj, 1)
	}
	fixed := dropExtraClosers(span)
	if fixed != span {
		if obj, ok := decodeObject(fixed); ok {
			return finishRepair(obj, 1)
		}
	}

	objs := collectObjects(s)
	if len(objs) == 0 {
		log.Printf("json ayrıştırılamadı neden=nesne_yok")
		if DebugDump {
			log.Printf("json kesit=%q", firstRunes(raw, 200))
		}
		return nil, ErrUnparseable
	}
	return finishRepair(mergeAll(objs), len(objs))
}

func finishRepair(merged map[string]any, nobj int) (map[string]any, error) {
	if len(merged) == 0 {
		log.Printf("json ayrıştırılamadı neden=bos_nesne")
		return nil, ErrUnparseable
	}
	normalizeJSONTypes(merged)
	log.Printf("json onarıldı nesne=%d alan=%d", nobj, len(merged))
	return merged, nil
}

func objectSpan(s string) string {
	i := strings.Index(s, "{")
	j := strings.LastIndex(s, "}")
	if i < 0 || j <= i {
		return ""
	}
	return s[i : j+1]
}

func decodeObject(s string) (map[string]any, bool) {
	if s == "" {
		return nil, false
	}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil || m == nil {
		return nil, false
	}
	rest := ""
	if n := int(dec.InputOffset()); n >= 0 && n < len(s) {
		rest = strings.TrimSpace(s[n:])
	}
	if rest != "" {
		return nil, false
	}
	return m, true
}

// dropExtraClosers kök nesneyi erken kapatan veya derinliği eksiye
// düşüren `}` karakterlerini siler. Dize içindeki parantezlere dokunmaz.
func dropExtraClosers(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	depth := 0
	inStr := false
	esc := false
	last := len(s) - 1
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			b.WriteByte(c)
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
			b.WriteByte(c)
		case '{':
			depth++
			b.WriteByte(c)
		case '}':
			next := depth - 1
			if next < 1 && i < last {
				continue
			}
			if next < 0 {
				continue
			}
			depth = next
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func stripMarkdown(s string) string {
	trimmed := strings.TrimSpace(s)
	lower := strings.ToLower(trimmed)
	if !strings.Contains(lower, "```") {
		return s
	}
	start := strings.Index(lower, "```")
	if start < 0 {
		return s
	}
	rest := trimmed[start+3:]
	if strings.HasPrefix(strings.ToLower(rest), "json") {
		rest = rest[4:]
	}
	rest = strings.TrimLeft(rest, " \t\r\n")
	end := strings.Index(rest, "```")
	if end < 0 {
		return rest
	}
	inner := strings.TrimSpace(rest[:end])
	if strings.Contains(inner, "{") {
		return inner
	}
	return s
}

func collectObjects(s string) []map[string]any {
	var objs []map[string]any
	rest := s
	for {
		rest = skipWS(rest)
		idx := strings.Index(rest, "{")
		if idx < 0 {
			break
		}
		rest = rest[idx:]
		dec := json.NewDecoder(strings.NewReader(rest))
		dec.UseNumber()
		var m map[string]any
		if err := dec.Decode(&m); err == nil && m != nil {
			objs = append(objs, m)
			n := int(dec.InputOffset())
			if n <= 0 || n > len(rest) {
				break
			}
			rest = rest[n:]
			continue
		}
		obj, _, ok := parseJSONObject(rest)
		if ok && len(obj) > 0 {
			objs = append(objs, obj)
		}
		break
	}
	return objs
}

func mergeAll(objs []map[string]any) map[string]any {
	out := make(map[string]any)
	for _, o := range objs {
		mergeMaps(out, o)
	}
	return out
}

func mergeMaps(dst, src map[string]any) {
	for k, sv := range src {
		dv, ok := dst[k]
		if !ok {
			dst[k] = sv
			continue
		}
		dm, dok := dv.(map[string]any)
		sm, sok := sv.(map[string]any)
		if dok && sok {
			mergeMaps(dm, sm)
			continue
		}
		da, dok := dv.([]any)
		sa, sok := sv.([]any)
		if dok && sok {
			dst[k] = append(append([]any{}, da...), sa...)
			continue
		}
		dst[k] = sv
	}
}

func skipWS(s string) string {
	return strings.TrimLeftFunc(s, unicode.IsSpace)
}

func parseJSONObject(s string) (map[string]any, string, bool) {
	s = skipWS(s)
	if s == "" || s[0] != '{' {
		return nil, s, false
	}
	rest := skipWS(s[1:])
	m := make(map[string]any)
	if rest != "" && rest[0] == '}' {
		return m, rest[1:], true
	}
	for {
		rest = skipWS(rest)
		if rest == "" {
			return m, rest, true
		}
		if rest[0] == '}' {
			return m, rest[1:], true
		}
		if rest[0] != '"' {
			return m, rest, true
		}
		key, rest2, ok := parseJSONString(rest)
		if !ok {
			return m, rest, true
		}
		rest2 = skipWS(rest2)
		if rest2 == "" || rest2[0] != ':' {
			return m, rest, true
		}
		rest2 = skipWS(rest2[1:])
		if rest2 == "" {
			return m, rest, true
		}
		val, rest3, ok := parseJSONValue(rest2)
		if !ok {
			return m, rest, true
		}
		m[key] = val
		rest = skipWS(rest3)
		if rest == "" {
			return m, rest, true
		}
		if rest[0] == ',' {
			rest = rest[1:]
			continue
		}
		if rest[0] == '}' {
			return m, rest[1:], true
		}
		return m, rest, true
	}
}

func parseJSONArray(s string) ([]any, string, bool) {
	s = skipWS(s)
	if s == "" || s[0] != '[' {
		return nil, s, false
	}
	rest := skipWS(s[1:])
	var arr []any
	if rest != "" && rest[0] == ']' {
		return arr, rest[1:], true
	}
	for {
		rest = skipWS(rest)
		if rest == "" {
			return arr, rest, true
		}
		if rest[0] == ']' {
			return arr, rest[1:], true
		}
		val, rest2, ok := parseJSONValue(rest)
		if !ok {
			return arr, rest, true
		}
		arr = append(arr, val)
		rest = skipWS(rest2)
		if rest == "" {
			return arr, rest, true
		}
		if rest[0] == ',' {
			rest = rest[1:]
			continue
		}
		if rest[0] == ']' {
			return arr, rest[1:], true
		}
		return arr, rest, true
	}
}

func parseJSONValue(s string) (any, string, bool) {
	s = skipWS(s)
	if s == "" {
		return nil, s, false
	}
	switch s[0] {
	case '{':
		return parseJSONObject(s)
	case '[':
		return parseJSONArray(s)
	case '"':
		return parseJSONString(s)
	case 't':
		if strings.HasPrefix(s, "true") {
			return true, s[4:], true
		}
	case 'f':
		if strings.HasPrefix(s, "false") {
			return false, s[5:], true
		}
	case 'n':
		if strings.HasPrefix(s, "null") {
			return nil, s[4:], true
		}
	default:
		if s[0] == '-' || (s[0] >= '0' && s[0] <= '9') {
			return parseJSONNumber(s)
		}
	}
	return nil, s, false
}

func parseJSONString(s string) (string, string, bool) {
	if s == "" || s[0] != '"' {
		return "", s, false
	}
	var b strings.Builder
	esc := false
	for i := 1; i < len(s); i++ {
		c := s[i]
		if esc {
			switch c {
			case '"', '\\', '/':
				b.WriteByte(c)
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'u':
				if i+4 >= len(s) {
					return "", s, false
				}
				r, err := unescapeU4(s[i+1 : i+5])
				if err != nil {
					return "", s, false
				}
				b.WriteRune(r)
				i += 4
			default:
				b.WriteByte(c)
			}
			esc = false
			continue
		}
		if c == '\\' {
			esc = true
			continue
		}
		if c == '"' {
			return b.String(), s[i+1:], true
		}
		b.WriteByte(c)
	}
	return "", s, false
}

func unescapeU4(h string) (rune, error) {
	var s string
	if err := json.Unmarshal([]byte(`"\u`+h+`"`), &s); err != nil {
		return 0, err
	}
	for _, rr := range s {
		return rr, nil
	}
	return 0, errors.New("bos")
}

func parseJSONNumber(s string) (json.Number, string, bool) {
	i := 0
	if s[0] == '-' {
		i++
		if i >= len(s) {
			return "", s, false
		}
	}
	if i >= len(s) || s[i] < '0' || s[i] > '9' {
		return "", s, false
	}
	if s[i] == '0' {
		i++
	} else {
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
	}
	if i < len(s) && s[i] == '.' {
		i++
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			return "", s, false
		}
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			return "", s, false
		}
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
	}
	return json.Number(s[:i]), s[i:], true
}

func normalizeJSONTypes(v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			x[k] = convertJSON(val)
		}
	case []any:
		for i, val := range x {
			x[i] = convertJSON(val)
		}
	}
}

func convertJSON(v any) any {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return int(i)
		}
		f, err := x.Float64()
		if err == nil {
			return f
		}
		return x.String()
	case map[string]any:
		normalizeJSONTypes(x)
		return x
	case []any:
		normalizeJSONTypes(x)
		return x
	default:
		return v
	}
}

func firstRunes(s string, n int) string {
	if n <= 0 || s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n])
}
