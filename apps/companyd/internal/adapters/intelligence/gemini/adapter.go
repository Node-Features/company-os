// Package gemini implements ports.ProviderAdapter against Google Gemini
// (gemini-2.5-flash — a fast, low-cost model matching the
// bounded-text-generation CapabilityDefinition). Swapped in for the
// internal/adapters/intelligence/anthropic adapter per user direction;
// both satisfy the same provider-independent ports.ProviderAdapter port
// (ADR-0003), so this required no change to Kernel, Governance,
// Application, or Runtime.
package gemini

import (
	"context"
	"errors"

	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"google.golang.org/genai"
)

const modelID = "gemini-2.5-flash"

// Adapter implements ports.ProviderAdapter.
type Adapter struct {
	client *genai.Client
}

// New builds an Adapter reading its API key from apiKey (populate from
// GEMINI_API_KEY).
func New(ctx context.Context, apiKey string) (*Adapter, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return &Adapter{client: client}, nil
}

func (a *Adapter) Generate(ctx context.Context, req ports.IntelligenceRequest) (ports.IntelligenceResult, error) {
	maxTokens := int32(req.MaxOutputTokens)
	if maxTokens <= 0 {
		maxTokens = 256
	}

	resp, err := a.client.Models.GenerateContent(ctx, modelID,
		[]*genai.Content{genai.NewContentFromText(req.Prompt, genai.RoleUser)},
		&genai.GenerateContentConfig{MaxOutputTokens: maxTokens},
	)
	if err != nil {
		return ports.IntelligenceResult{}, classify(err)
	}

	result := ports.IntelligenceResult{
		Text:     resp.Text(),
		ModelID:  modelID,
		Provider: "gemini",
	}
	if len(resp.Candidates) > 0 {
		result.FinishReason = string(resp.Candidates[0].FinishReason)
	}
	if resp.UsageMetadata != nil {
		result.Usage.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
		result.Usage.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	}
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

// classify maps genai.APIError HTTP status codes to retryable (429 rate
// limit, 5xx server error) vs terminal (400 invalid request, 401/403
// auth/permission, 404 not found), per runtime.md#failure-semantics.
func classify(err error) error {
	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Code == 429 || apiErr.Code >= 500 {
			return &classifiedError{err: err, retryable: true}
		}
		return &classifiedError{err: err, retryable: false}
	}
	// Network errors, context deadline exceeded, etc. — no HTTP response
	// at all — are treated as retryable: the request may not have reached
	// the provider.
	return &classifiedError{err: err, retryable: true}
}
