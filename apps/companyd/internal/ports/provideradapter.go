// ProviderAdapter port for Intelligence capability dispatch.
// See docs/architecture/intelligence.md.
package ports

import "context"

// IntelligenceRequest is the bounded-text-generation CapabilityDefinition's
// input contract (first-slice plan, fixtures.Registry.Capability()).
type IntelligenceRequest struct {
	Prompt          string
	MaxOutputTokens int
}

// IntelligenceResult is the bounded-text-generation CapabilityDefinition's
// output contract.
type IntelligenceResult struct {
	Text    string
	ModelID string
	// Provider names which concrete adapter served the request — set by
	// each of internal/adapters/intelligence/{gemini,openai,anthropic}, so
	// a caller composing multiple adapters
	// (internal/adapters/intelligence/fallback) doesn't have to guess
	// which one won.
	Provider     string
	FinishReason string
	Usage        struct {
		InputTokens  int
		OutputTokens int
	}
}

// ProviderAdapter is the one first-slice IntelligenceCapability's dispatch
// port. Three implementations exist under internal/adapters/intelligence/
// ({gemini,openai,anthropic}), all satisfying this same port —
// demonstrating the provider-independence ADR-0003 requires.
// internal/adapters/intelligence/fallback composes any number of them into
// one ProviderAdapter that tries providers in priority order and skips a
// recently-failed one for a cooldown window; that composed adapter is what
// cmd/companyd/main.go actually wires into Runtime.
type ProviderAdapter interface {
	Generate(ctx context.Context, req IntelligenceRequest) (IntelligenceResult, error)
}

// classified is the duck-typed shape every adapter's error-classification
// wrapper satisfies (each internal/adapters/intelligence/* package defines
// its own unexported classifiedError with a Retryable() bool method).
type classified interface{ Retryable() bool }

// IsRetryable reports whether err was classified retryable by whichever
// ProviderAdapter produced it (docs/architecture/runtime.md#failure-semantics:
// rate-limit/timeout/outage vs invalid-request/auth). An unclassified error
// is treated as terminal — fail closed on retries/fallback rather than
// looping on an error nobody vouched for.
func IsRetryable(err error) bool {
	c, ok := err.(classified)
	return ok && c.Retryable()
}
