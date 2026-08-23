// Package knowledge is Knowledge's Kernel-equivalent: pure legality
// functions only, no I/O. See docs/architecture/knowledge.md,
// docs/domain/knowledge.md. Lives under internal/kernel (like
// internal/kernel/objective), not internal/departments — Knowledge is a
// cross-cutting architecture concern, not a business department.
package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// ContentDigest is the canonical content/source digest for a KnowledgeItem's
// Claim (docs/domain/knowledge.md: "canonical content/source digest"). A
// plain SHA-256 of the exact bounded claim text — no canonical-JSON step is
// needed here, unlike kernel/workflow's and kernel/objective's digest
// helpers, since a Claim is already a single string with no key ordering to
// normalize.
func ContentDigest(claim string) string {
	sum := sha256.Sum256([]byte(claim))
	return hex.EncodeToString(sum[:])
}

// proposalDigest is a self-contained copy of kernel/objective/digest.go's
// canonicalDigest — small and pure enough that duplicating it here avoids
// importing one Kernel package from another for a single helper. Distinct
// purpose from ContentDigest above: this canonicalizes a whole
// proposal/command structure (for ProposalDigest/CommandDigest, review.go),
// not a single claim string.
func proposalDigest(v any) string {
	b := canonicalJSON(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func canonicalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic("kernel/knowledge: canonicalJSON: " + err.Error())
	}
	var generic any
	if err := json.Unmarshal(b, &generic); err != nil {
		panic("kernel/knowledge: canonicalJSON: " + err.Error())
	}
	sorted, err := json.Marshal(sortKeys(generic))
	if err != nil {
		panic("kernel/knowledge: canonicalJSON: " + err.Error())
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
