package agent

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	confidenceModel     = 0.9
	confidenceRepaired  = 0.6
	confidenceFilled    = 0.2
	minSearchValueRunes = 3
)

var coreFields = []coreField{
	{Schema: "donem", GraphQL: "donem"},
	{Schema: "oda_kontenjanlari", GraphQL: "odaKontenjanlari"},
	{Schema: "fiyatlar", GraphQL: "fiyatlar"},
	{Schema: "release", GraphQL: "release"},
	{Schema: "stop_sale", GraphQL: "stopSale"},
}

type coreField struct {
	Schema  string
	GraphQL string
}

// FieldMeta bir çekirdek alanın güven skoru ve kaynak sayfasıdır.
type FieldMeta struct {
	FieldPath  string
	SchemaPath string
	Confidence float64
	SourcePage *int32
}

// BuildExtractionMeta çekirdek alanlar için güven skoru ve kaynak sayfa üretir.
func BuildExtractionMeta(data map[string]any, repairs []string, pages []string) []FieldMeta {
	out := make([]FieldMeta, 0, len(coreFields))
	for _, f := range coreFields {
		conf := confidenceFor(f.Schema, repairs)
		page := sourcePage(data[f.Schema], pages)
		out = append(out, FieldMeta{
			FieldPath:  f.GraphQL,
			SchemaPath: f.Schema,
			Confidence: conf,
			SourcePage: page,
		})
	}
	return out
}

func confidenceFor(field string, repairs []string) float64 {
	filled := false
	touched := false
	for _, note := range repairs {
		if noteMentions(note, field) {
			if strings.HasPrefix(note, "zorunlu alan dolduruldu:") {
				filled = true
			} else {
				touched = true
			}
		}
	}
	if field == "stop_sale" {
		for _, note := range repairs {
			if strings.Contains(note, "stop_sale donem içinden üst düzeye taşındı") {
				touched = true
			}
		}
	}
	switch {
	case filled:
		return confidenceFilled
	case touched:
		return confidenceRepaired
	default:
		return confidenceModel
	}
}

func noteMentions(note, field string) bool {
	idx := strings.LastIndex(note, ": ")
	path := note
	if idx >= 0 {
		path = strings.TrimSpace(note[idx+2:])
	}
	if path == field {
		return true
	}
	return strings.HasPrefix(path, field+".") || strings.HasPrefix(path, field+"[")
}

func sourcePage(value any, pages []string) *int32 {
	needles := collectNeedles(value)
	for i, page := range pages {
		lower := strings.ToLower(page)
		for _, n := range needles {
			if strings.Contains(lower, n) {
				p := int32(i + 1)
				return &p
			}
		}
	}
	return nil
}

func collectNeedles(v any) []string {
	var out []string
	walkNeedles(v, &out)
	return out
}

func walkNeedles(v any, out *[]string) {
	switch x := v.(type) {
	case nil:
		return
	case string:
		addNeedle(x, out)
	case fmt.Stringer:
		addNeedle(x.String(), out)
	case int:
		addNeedle(fmt.Sprintf("%d", x), out)
	case int32:
		addNeedle(fmt.Sprintf("%d", x), out)
	case int64:
		addNeedle(fmt.Sprintf("%d", x), out)
	case float64:
		if x == float64(int64(x)) {
			addNeedle(fmt.Sprintf("%d", int64(x)), out)
		} else {
			addNeedle(fmt.Sprintf("%v", x), out)
		}
	case []any:
		for _, item := range x {
			walkNeedles(item, out)
		}
	case map[string]any:
		for _, item := range x {
			walkNeedles(item, out)
		}
	default:
		addNeedle(fmt.Sprintf("%v", x), out)
	}
}

func addNeedle(s string, out *[]string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	n := 0
	for _, r := range s {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	if n < minSearchValueRunes {
		return
	}
	*out = append(*out, strings.ToLower(s))
}
