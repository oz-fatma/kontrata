package agent

import (
	"os"
	"strings"
)

const (
	ExtractModeSingle  = "single"
	ExtractModeChunked = "chunked"
)

// ExtractModeFromEnv EXTRACT_MODE ortam değişkenini okur; varsayılan single.
func ExtractModeFromEnv() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("EXTRACT_MODE"))) {
	case ExtractModeChunked:
		return ExtractModeChunked
	default:
		return ExtractModeSingle
	}
}
