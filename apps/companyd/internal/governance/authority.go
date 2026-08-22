package governance

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
)

// Authority is the governed-decision contract Application depends on before
// any Kernel-authorized effect proceeds. The package function Evaluate
// already has this exact signature today — Authorize is a naming choice for
// the interface, not a behavior change; nothing here is a new
// implementation.
//
// It has only one method, Authorize, not Grant or Revoke: per
// docs/architecture/governance.md, Governance evaluates a request against
// Authority that was already resolved elsewhere — "Authority | Active
// delegation that bounds what the principal may request" is listed as an
// INPUT to the decision (carried in Request's PrincipalID/EvidencePresent
// fields this slice), never something Governance itself mutates. Granting
// or revoking delegated authority is a Principal/Identity-domain concern
// (docs/domain/principal.md's "delegation references") with no Go
// implementation anywhere in this codebase yet — adding Grant/Revoke here
// would invent an API with nothing behind it.
//
// Contract: Authorize returns exactly one of ALLOW, DENY, or
// REQUIRE_APPROVAL (policy.Decision) for a legitimately evaluated request —
// the error return is reserved for infrastructure failure (e.g. a policy
// store that could not be read), which callers must treat as fail-closed,
// never as an implicit DENY already decided. Authorize does NOT guarantee
// its result remains valid at execution time — per governance.md step 8,
// dispatch-time re-evaluation is a second, separate Authorize call the
// caller must make immediately before an external effect; this method does
// not do that internally. Authorize does NOT guarantee ordering or mutual
// exclusion between concurrent calls evaluating the same resource — two
// concurrent calls for the same proposal may both return ALLOW; the
// resulting write conflict is caught downstream by Persistence's
// compare-and-write, not by this method. REQUIRE_APPROVAL does NOT mean
// eventually-ALLOW: it means the request is otherwise eligible but lacks
// valid Approval evidence right now, and must never be treated as a weak
// allow by a caller.
type Authority interface {
	Authorize(ctx context.Context, req Request) (policy.GovernanceDecision, error)
}
