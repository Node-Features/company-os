package observability

// maxProviderErrorLen bounds how much of a provider error message is ever
// logged — defense against a provider echoing back a large payload (e.g. a
// safety-filter or rate-limit message that itself quotes prompt content)
// inside its error string.
const maxProviderErrorLen = 200

// SafeProviderError renders a provider adapter error for logging without
// ever emitting a raw, unbounded provider response body — closes the gap
// found in internal/adapters/intelligence/fallback's prior %v-formatted
// provider-error logging. retryable is whatever ports.IsRetryable already
// classified the error as, so this never re-derives that decision. See
// docs/architecture/observability.md's Redaction section.
func SafeProviderError(err error, retryable bool) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) > maxProviderErrorLen {
		msg = msg[:maxProviderErrorLen] + "...(truncated)"
	}
	kind := "terminal"
	if retryable {
		kind = "retryable"
	}
	return kind + ": " + msg
}
