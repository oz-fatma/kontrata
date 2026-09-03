// Package agent Okuyucu çıkarım motorunu çalıştırır.
package agent

import (
	"time"
)

// ExtractResult bir sözleşme metninden üretilen şema çıktısıdır.
type ExtractResult struct {
	Data         map[string]any
	Repairs      []string
	SchemaErrors []string
	RetryCount   int
	Duration     time.Duration
	Meta         []FieldMeta
}
