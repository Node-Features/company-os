// Package openai implements ports.ProviderAdapter against OpenAI
// (gpt-5-nano — a fast, low-cost model matching the
// bounded-text-generation CapabilityDefinition), as an alternative to
// internal/adapters/intelligence/{anthropic,gemini}. All three satisfy the
// same provider-independent ports.ProviderAdapter port (ADR-0003); only
// one is wired into cmd/companyd/main.go at a time.
package openai

import (
	"context"
	"errors"

	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

// Adapter implements ports.ProviderAdapter.
type Adapter struct {
	client openai.Client
}

// New builds an Adapter reading its API key from apiKey (populate from
// OPENAI_API_KEY).
func New(apiKey string) *Adapter {
	return &Adapter{client: openai.NewClient(option.WithAPIKey(apiKey))}
}

func (a *Adapter) Generate(ctx context.Context, req ports.IntelligenceRequest) (ports.IntelligenceResult, error) {
	maxTokens := int64(req.MaxOutputTokens)
	if maxTokens <= 0 {
		maxTokens = 256
	}

	resp, err := a.client.Responses.New(ctx, responses.ResponseNewParams{
		Model:           openai.ChatModelGPT5Nano,
		Input:           responses.ResponseNewParamsInputUnion{OfString: openai.String(req.Prompt)},
		MaxOutputTokens: openai.Int(maxTokens),
	})
	if err != nil {
		return ports.IntelligenceResult{}, classify(err)
	}

	result := ports.IntelligenceResult{
		Text:     resp.OutputText(),
		ModelID:  string(resp.Model),
		Provider: "openai",
	}
	result.Usage.InputTokens = int(resp.Usage.InputTokens)
	result.Usage.OutputTokens = int(resp.Usage.OutputTokens)
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

// classify maps openai.Error HTTP status codes to retryable (429 rate
// limit, 5xx server error) vs terminal (400 invalid request, 401/403
// auth/permission, 404 not found), per runtime.md#failure-semantics.
func classify(err error) error {
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 429 || apiErr.StatusCode >= 500 {
			return &classifiedError{err: err, retryable: true}
		}
		return &classifiedError{err: err, retryable: false}
	}
	// Network errors, context deadline exceeded, etc. — no HTTP response
	// at all — are treated as retryable: the request may not have reached
	// the provider.
	return &classifiedError{err: err, retryable: true}
}
