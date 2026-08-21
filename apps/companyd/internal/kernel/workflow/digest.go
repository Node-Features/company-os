package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// canonicalDigest implements first-slice plan decision #10: canonical JSON
// (map keys sorted, no floating whitespace) hashed with SHA-256, hex
// encoded. Used for both the proposal digest and the trusted-context
// digest (docs/domain/command.md's open question on serialization).
func canonicalDigest(v any) string {
	b := canonicalJSON(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// canonicalJSON re-marshals v through a generic interface{} so map keys are
// always sorted (encoding/json already sorts map[string]any keys; this
// documents that the digest depends on it and gives one seam to swap in a
// stricter canonical form later).
func canonicalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		// Kernel functions do no I/O and never fail on well-formed inputs
		// constructed from this package's own types; a marshal error here
		// means a caller passed an unmarshalable value, a programming error.
		panic("kernel/workflow: canonicalJSON: " + err.Error())
	}
	var generic any
	if err := json.Unmarshal(b, &generic); err != nil {
		panic("kernel/workflow: canonicalJSON: " + err.Error())
	}
	sorted, err := json.Marshal(sortKeys(generic))
	if err != nil {
		panic("kernel/workflow: canonicalJSON: " + err.Error())
	}
	return sorted
}

func sortKeys(v any) any {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(t))
		for _, k := range keys {
			out[k] = sortKeys(t[k])
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = sortKeys(e)
		}
		return out
	default:
		return v
	}
}
