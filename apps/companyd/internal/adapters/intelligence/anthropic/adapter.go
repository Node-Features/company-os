// Package anthropic implements ports.ProviderAdapter against Anthropic
// Claude (first-slice plan decision #1: claude-haiku-4-5, the cheapest and
// fastest currently available Claude model, non-streaming, matching the
// bounded-text-generation CapabilityDefinition).
package anthropic

import (
	"context"
	"errors"
	"strings"

	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Adapter implements ports.ProviderAdapter.
type Adapter struct {
	client anthropic.Client
}

// New builds an Adapter reading its API key from apiKey (populate from
// ANTHROPIC_API_KEY).
func New(apiKey string) *Adapter {
	return &Adapter{client: anthropic.NewClient(option.WithAPIKey(apiKey))}
}

func (a *Adapter) Generate(ctx context.Context, req ports.IntelligenceRequest) (ports.IntelligenceResult, error) {
	maxTokens := int64(req.MaxOutputTokens)
	if maxTokens <= 0 {
		maxTokens = 256
	}

	msg, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(req.Prompt)),
		},
	})
	if err != nil {
		return ports.IntelligenceResult{}, classify(err)
	}

	var text strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	result := ports.IntelligenceResult{
		Text:         text.String(),
		ModelID:      string(msg.Model),
		Provider:     "anthropic",
		FinishReason: string(msg.StopReason),
	}
	result.Usage.InputTokens = int(msg.Usage.InputTokens)
	result.Usage.OutputTokens = int(msg.Usage.OutputTokens)
	return result, nil
}

// classifiedError satisfies runtime's classifyRetryable(err) via a
// Retryable() bool method (docs/architecture/runtime.md#failure-semantics).
type classifiedError struct {
	err       error
	retryable bool
}

func (c *classifiedError) Error() string   { return c.err.Error() }
func (c *classifiedError) Unwrap() error   { return c.err }
func (c *classifiedError) Retryable() bool { return c.retryable }

// classify maps Anthropic SDK errors to retryable (timeout/rate_limit/
// overloaded/server error) vs terminal (invalid_request/auth/permission/
// not_found/billing), per runtime.md#failure-semantics.
func classify(err error) error {
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		switch apiErr.Type() {
		case anthropic.ErrorTypeRateLimitError, anthropic.ErrorTypeOverloadedError, anthropic.ErrorTypeTimeoutError, anthropic.ErrorTypeAPIError:
			return &classifiedError{err: err, retryable: true}
		case anthropic.ErrorTypeInvalidRequestError, anthropic.ErrorTypeAuthenticationError, anthropic.ErrorTypePermissionError, anthropic.ErrorTypeNotFoundError, anthropic.ErrorTypeBillingError:
			return &classifiedError{err: err, retryable: false}
		}
		if apiErr.StatusCode >= 500 || apiErr.StatusCode == 429 {
			return &classifiedError{err: err, retryable: true}
		}
		return &classifiedError{err: err, retryable: false}
	}
	// Network errors, context deadline exceeded, etc. — no HTTP response
	// at all — are treated as retryable: the request may not have reached
	// the provider.
	return &classifiedError{err: err, retryable: true}
}
