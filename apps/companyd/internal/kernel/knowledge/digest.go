// Package knowledge is Knowledge's Kernel-equivalent: pure legality
// functions only, no I/O. See docs/architecture/knowledge.md,
// docs/domain/knowledge.md. Lives under internal/kernel (like
// internal/kernel/objective), not internal/departments — Knowledge is a
// cross-cutting architecture concern, not a business department.
package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
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
