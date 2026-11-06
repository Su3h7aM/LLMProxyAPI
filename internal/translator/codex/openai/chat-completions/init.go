package chat_completions

import (
	. "github.com/Su3h7aM/LLMProxyAPI/internal/constant"
	"github.com/Su3h7aM/LLMProxyAPI/internal/interfaces"
	"github.com/Su3h7aM/LLMProxyAPI/internal/translator/translator"
)

func init() {
	translator.Register(
		OpenAI,
		Codex,
		ConvertOpenAIRequestToCodex,
		interfaces.TranslateResponse{
			Stream:    ConvertCodexResponseToOpenAI,
			NonStream: ConvertCodexResponseToOpenAINonStream,
		},
	)
}
