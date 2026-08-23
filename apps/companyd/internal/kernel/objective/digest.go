package objective

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// canonicalDigest is a self-contained copy of
// internal/kernel/workflow/digest.go's function of the same name — small
// and pure enough that duplicating it here avoids importing one Kernel
// package from another for a single helper. See that file's doc comment
// for the algorithm (canonical JSON, sorted map keys, SHA-256, hex).
func canonicalDigest(v any) string {
	b := canonicalJSON(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func canonicalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic("kernel/objective: canonicalJSON: " + err.Error())
	}
	var generic any
	if err := json.Unmarshal(b, &generic); err != nil {
		panic("kernel/objective: canonicalJSON: " + err.Error())
	}
	sorted, err := json.Marshal(sortKeys(generic))
	if err != nil {
		panic("kernel/objective: canonicalJSON: " + err.Error())
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
