package llm

// BuildChatPrompt Qwen sohbet şablonunu kurar.
// Model bu şablonla eğitildi; ham metin gönderilirse çıktı bozulur.
// Son satırdan sonra yeni satır yoktur — model assistant konumundan devam eder.
func BuildChatPrompt(systemPrompt, userPrompt string) string {
	return "<|im_start|>system\n" +
		systemPrompt +
		"<|im_end|>\n" +
		"<|im_start|>user\n" +
		userPrompt +
		"<|im_end|>\n" +
		"<|im_start|>assistant"
}
