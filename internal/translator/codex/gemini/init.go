package gemini

import (
	. "github.com/router-for-me/LLMProxyAPI/v8/internal/constant"
	"github.com/router-for-me/LLMProxyAPI/v8/internal/interfaces"
	"github.com/router-for-me/LLMProxyAPI/v8/internal/translator/translator"
)

func init() {
	translator.Register(
		Gemini,
		Codex,
		ConvertGeminiRequestToCodex,
		interfaces.TranslateResponse{
			Stream:     ConvertCodexResponseToGemini,
			NonStream:  ConvertCodexResponseToGeminiNonStream,
			TokenCount: GeminiTokenCount,
		},
	)
}
