package chat_completions

import (
	. "github.com/Su3h7aM/LLMProxyAPI/internal/constant"
	"github.com/Su3h7aM/LLMProxyAPI/internal/interfaces"
	"github.com/Su3h7aM/LLMProxyAPI/internal/translator/translator"
)

func init() {
	translator.Register(
		OpenAI,
		Antigravity,
		ConvertOpenAIRequestToAntigravity,
		interfaces.TranslateResponse{
			Stream:    ConvertAntigravityResponseToOpenAI,
			NonStream: ConvertAntigravityResponseToOpenAINonStream,
		},
	)
}
