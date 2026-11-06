package responses

import (
	. "github.com/Su3h7aM/LLMProxyAPI/internal/constant"
	"github.com/Su3h7aM/LLMProxyAPI/internal/interfaces"
	"github.com/Su3h7aM/LLMProxyAPI/internal/translator/translator"
)

func init() {
	translator.Register(
		OpenaiResponse,
		Interactions,
		ConvertOpenAIResponsesRequestToInteractions,
		interfaces.TranslateResponse{
			Stream:    ConvertInteractionsResponseToOpenAIResponses,
			NonStream: ConvertInteractionsResponseToOpenAIResponsesNonStream,
		},
	)
	translator.Register(
		Interactions,
		OpenaiResponse,
		ConvertInteractionsRequestToOpenAIResponses,
		interfaces.TranslateResponse{
			Stream:    ConvertOpenAIResponsesResponseToInteractions,
			NonStream: ConvertOpenAIResponsesResponseToInteractionsNonStream,
		},
	)
}
