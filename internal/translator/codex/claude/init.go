package claude

import (
	"context"

	. "github.com/Su3h7aM/LLMProxyAPI/internal/constant"
	"github.com/Su3h7aM/LLMProxyAPI/internal/interfaces"
	"github.com/Su3h7aM/LLMProxyAPI/internal/translator/translator"
	sdktranslator "github.com/Su3h7aM/LLMProxyAPI/sdk/translator"
)

func init() {
	translator.Register(
		Claude,
		Codex,
		ConvertClaudeRequestToCodex,
		interfaces.TranslateResponse{
			Stream:     ConvertCodexResponseToClaude,
			NonStream:  ConvertCodexResponseToClaudeNonStream,
			TokenCount: ClaudeTokenCount,
		},
	)
	sdktranslator.RegisterRequestEnvelope(sdktranslator.FormatClaude, sdktranslator.FormatCodex,
		func(ctx context.Context, req sdktranslator.RequestEnvelope) sdktranslator.RequestEnvelope {
			req.Body, req.Err = ConvertClaudeRequestToCodexContext(ctx, req.Model, req.Body, req.Stream)
			return req
		})
}
