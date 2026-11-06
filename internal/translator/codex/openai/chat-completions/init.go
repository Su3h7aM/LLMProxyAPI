package chat_completions

import (
	. "github.com/router-for-me/LLMProxyAPI/v8/internal/constant"
	"github.com/router-for-me/LLMProxyAPI/v8/internal/interfaces"
	"github.com/router-for-me/LLMProxyAPI/v8/internal/translator/translator"
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
