package responses

import (
	. "github.com/Su3h7aM/LLMProxyAPI/internal/constant"
	"github.com/Su3h7aM/LLMProxyAPI/internal/interfaces"
	"github.com/Su3h7aM/LLMProxyAPI/internal/translator/translator"
	sdktranslator "github.com/Su3h7aM/LLMProxyAPI/sdk/translator"
)

func init() {
	translator.Register(
		OpenaiResponse,
		Claude,
		nil,
		interfaces.TranslateResponse{
			Stream:    ConvertClaudeResponseToOpenAIResponses,
			NonStream: ConvertClaudeResponseToOpenAIResponsesNonStream,
		},
	)
	// The envelope form receives the request context, which carries the
	// upstream.claude.tool-search value the executor pinned for the request.
	sdktranslator.RegisterRequestEnvelope(
		sdktranslator.FormatOpenAIResponse,
		sdktranslator.FormatClaude,
		ConvertOpenAIResponsesRequestEnvelopeToClaude,
	)
}
