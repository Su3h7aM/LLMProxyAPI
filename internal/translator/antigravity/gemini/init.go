package gemini

import (
	. "github.com/Su3h7aM/LLMProxyAPI/internal/constant"
	"github.com/Su3h7aM/LLMProxyAPI/internal/interfaces"
	"github.com/Su3h7aM/LLMProxyAPI/internal/translator/translator"
)

func init() {
	translator.Register(
		Gemini,
		Antigravity,
		ConvertGeminiRequestToAntigravity,
		interfaces.TranslateResponse{
			Stream:     ConvertAntigravityResponseToGemini,
			NonStream:  ConvertAntigravityResponseToGeminiNonStream,
			TokenCount: GeminiTokenCount,
		},
	)
}
