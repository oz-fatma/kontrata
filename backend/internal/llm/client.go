// Package llm HuggingFace Inference Endpoint üzerinden metin üretir.
// Gönderilen prompt ve model çıktısı loglanmaz.
package llm

import "context"

// Client bir sistem ve kullanıcı mesajından düz metin üretir.
type Client interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}
