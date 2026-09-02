package extract

import (
	"bytes"
	_ "embed"
	"log"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed kontrat.json
var schemaJSON []byte

var (
	schemaOnce sync.Once
	compiled   *jsonschema.Schema
	schemaErr  error
)

func loadSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		c := jsonschema.NewCompiler()
		c.AssertFormat()
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
		if err != nil {
			schemaErr = err
			return
		}
		id := "https://github.com/oz-fatma/kontrata/ml/schema/kontrat.json"
		if err := c.AddResource(id, doc); err != nil {
			schemaErr = err
			return
		}
		sch, err := c.Compile(id)
		if err != nil {
			schemaErr = err
			return
		}
		compiled = sch
	})
	return compiled, schemaErr
}

// Validate jsonschema hatalarını alan yolu olarak döner. Sözleşme değeri içermez.
func Validate(data map[string]any) []string {
	sch, err := loadSchema()
	if err != nil {
		log.Printf("sema derlenemedi: %v", err)
		return []string{"şema derlenemedi"}
	}
	err = sch.Validate(data)
	if err == nil {
		return nil
	}
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		log.Printf("sema dogrulama hata=1")
		return []string{"şema doğrulaması başarısız"}
	}
	msgs := flattenErrors(ve)
	log.Printf("sema dogrulama hata=%d", len(msgs))
	return msgs
}

func flattenErrors(e *jsonschema.ValidationError) []string {
	out := e.BasicOutput()
	var msgs []string
	if out != nil {
		collectUnits(*out, &msgs)
	}
	if len(msgs) == 0 {
		return []string{"şema doğrulaması başarısız"}
	}
	return msgs
}

func collectUnits(u jsonschema.OutputUnit, msgs *[]string) {
	if u.Error != nil && len(u.Errors) == 0 {
		loc := pointerToPath(u.InstanceLocation)
		kw := u.KeywordLocation
		if loc == "" {
			loc = "(kök)"
		}
		msg := loc
		if kw != "" {
			msg = loc + " " + kw
		}
		*msgs = append(*msgs, msg)
	}
	for _, child := range u.Errors {
		collectUnits(child, msgs)
	}
}

func pointerToPath(p string) string {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return ""
	}
	return strings.ReplaceAll(p, "/", ".")
}

// SchemaJSON gömülü kontrat şemasıdır. Test ve gömme doğrulaması için.
func SchemaJSON() []byte {
	return schemaJSON
}
