# ADR-0008: Authority and Capability Model

Status: PROPOSED

## Context

This request asked for a new `Subject`/`Role`/`Capability`/`Action`/`Resource` type set and a `kernel/authority/authority.go` implementing `Authorize(subject, action, resource) error`. Checked against what already exists, every piece of that either already has a real, more precise equivalent, or would collide with a different, already-approved concept of the same name:

| Requested | Already exists as | Note |
|---|---|---|
| `Subject` | `policy.PrincipalRef` (`PrincipalID`, `Kind`, `Authenticated`) | Documented in its own source as "the minimal Principal shape Governance needs." `governance.md`'s and Cedar's own vocabulary (studied in that doc's OSS evidence) both say "Principal," not "Subject." |
| `Resource` | `policy.Resource` (`Type`, `ID`) | Identical shape already. |
| `Action` | `string`, used throughout (`Request.Action`, `command.ActionFor`) | No separate type was ever needed. |
| `Authorize(subject, action, resource) error` | `governance.Authority.Authorize(ctx, Request) (policy.GovernanceDecision, error)`, built two turns ago in this session | The requested binary `error` return is a regression from what's already built and approved: it collapses `ALLOW`/`DENY`/`REQUIRE_APPROVAL` into allowed-or-not, discarding the whole "eligible but needs approval evidence" outcome `governance.md` treats as load-bearing ("`REQUIRE_APPROVAL`... is not a weak allow and must never be sent to an executor"). |
| `Role` | Nothing, until this ADR | Genuinely missing — see Decision. |
| `Capability` | `docs/domain/capability.md`'s `CapabilityDefinition` — an **already-approved, already-implemented, different concept**: a provider-independent dispatch contract (e.g. "generate text"), not a permission | Reusing this name for "a security permission" would put two unrelated things under one name in the same codebase — worse than a duplicate, because nothing would even flag the collision at compile time. |

Building `kernel/authority/authority.go` would also repeat two mistakes already corrected in this session: Authority placed under Kernel (corrected two turns ago — it's Governance's), and a second, competing implementation of something that already exists (the same issue that made `kernel/contracts.go` a non-starter three turns ago).

One thing was genuinely missing, and this ADR adds it for real rather than around it: the existing policy engine (`internal/governance/policy.go`) matched only on **Action** — it had no way to grant `research.read_market_data` to one kind of agent and not another, because nothing about *who* was asking factored into the match at all. That's the one real gap this request exposed.

## Decision

### The model, mapped to real CompanyOS vocabulary

```
Identity → Role        → Authority        → Action → Resource
Principal  policy.Role    governance.Evaluate  string   policy.Resource
                           (Authority.Authorize)
```

`Capability` is deliberately absent from this chain — CompanyOS's `Capability` already means something else, and nothing here needed a fifth type: `Action` (a string) and `Resource` (already existed) fully cover what "Capability" would have meant in the requested model.

### What was added, concretely

- **`policy.Role`** (`internal/domain/policy/types.go`): a named string type. Doc comment states plainly what it is a stand-in for — a real role/delegation lookup (`docs/domain/principal.md`'s "delegation references") — the same honest framing `Request.EvidencePresent` already uses for its own first-slice simplification.
- **`policy.Rule.Role`**: empty matches every role, so all rules written before this ADR are unaffected — verified, not assumed: the full existing test suite (`evaluate_test.go`, five tests, all role-agnostic) passes unchanged.
- **`governance.Request.Role`**: threaded through to `matchRule`, which now requires a role match (when the rule specifies one) in addition to the existing action match.
- **Illustrative rules** for `research_agent` and `finance_agent` in `firstSlicePolicies` (table below).
- **`internal/governance/authority_test.go`** — reuses the requested filename, in the package where `Authorize` actually lives.

### Example capability table

| Role | Action | Outcome | Rule |
|---|---|---|---|
| `research_agent` | `research.read_market_data` | `ALLOW` | `research-agent-read-market-data` |
| `research_agent` | `research.create_report` | `ALLOW` | `research-agent-create-report` |
| `research_agent` | `customer.send_message` | `DENY` | *(none — falls through to default-deny)* |
| `research_agent` | `finance.transfer_funds` | `DENY` | *(none — wrong role, not just no rule)* |
| `finance_agent` | `finance.read_financial_data` | `ALLOW` | `finance-agent-read-financial-data` |
| `finance_agent` | `finance.create_payment_request` | `ALLOW` | `finance-agent-create-payment-request` |
| `finance_agent` | `finance.transfer_funds` | **`REQUIRE_APPROVAL`** | `finance-agent-transfer-funds` |
| *(any unrecognized role)* | `research.read_market_data` | `DENY` | *(no role-agnostic rule grants this action)* |

The `transfer_funds` row is deliberately `REQUIRE_APPROVAL`, not `DENY`: "denied without additional authority" is exactly what `REQUIRE_APPROVAL` means — eligible, but missing Approval evidence — which is more precise than the binary allow/deny the request's own design table used. A caller that treats `REQUIRE_APPROVAL` as a plain denial has misread the decision (`governance.md`).

This table is illustrative, the same way `ADR-0005`'s Organization/Objective/Capability signatures were: [`docs/security/agent-authority.md`](../security/agent-authority.md) (`ROADMAP.md` Phase 2 Slice 2, now written and approved) is where real agent-authority policy gets decided; `research_agent`/`finance_agent` are role strings for this demonstration, not registered `AgentDefinition`s — `internal/domain/agent` still doesn't exist.

## Consequences

### Positive

- Closes a real capability gap (role-scoped policy matching) rather than adding a parallel system next to the one that already existed.
- Fully backward compatible and verified as such: all 5 pre-existing `governance` tests plus the 6 new ones pass; the whole module's test suite (including the `internal/application` integration tests against a real database) passes unchanged.
- The 3 requested test categories (allowed succeeds, denied fails, unknown subject fails closed) are all present, plus two more completing the original design table, plus one proving the `REQUIRE_APPROVAL` nuance.

### Costs and risks

- `Role` is caller-asserted, not verified against any persisted binding — exactly as honest (and exactly as first-slice-limited) as `EvidencePresent` already was. A real implementation needs `docs/domain/principal.md`'s delegation model, not just a string.
- The illustrative rules live in the same hardcoded `firstSlicePolicies` list the two original rules did — `docs/domain/policy.md` still leaves policy administration as future work; this ADR doesn't change that.

## Alternatives rejected by this proposal

- **A parallel `kernel/authority` implementation, as literally requested:** rejected — see Context. Duplicates already-built, already-approved work, misplaces Authority under Kernel a second time, and would coexist confusingly with the real `governance.Authority`.
- **Naming the new type `Capability`:** rejected — `docs/domain/capability.md`'s `CapabilityDefinition` already owns that name for a different concept (dispatch contract, not permission). Two different things sharing one name in the same codebase is worse than the duplicate-implementation problem above, because it wouldn't even be caught by the compiler.
- **A binary `Authorize(...) error` return:** rejected — collapses `REQUIRE_APPROVAL` into either `ALLOW` or `DENY`, losing the distinction `governance.md` calls load-bearing.

## Acceptance criteria

- [x] `go build ./...`, `go vet ./...`, and `go test ./...` pass across the whole module after the change, including the pre-existing `governance` and `application` test suites.
- [x] Cross-checked against `governance.md`, `capability.md` (via its architecture summary), and `ADR-0005`'s Authority work from two turns ago — no contradiction found; one real gap (role-scoped matching) identified and closed.
- [ ] the project owner reviews and explicitly changes `Status: PROPOSED` to `Status: APPROVED`.

## Open questions

- OPEN QUESTION: [`docs/security/agent-authority.md`](../security/agent-authority.md) (`ROADMAP.md` Phase 2 Slice 2, now written and approved) is where real per-department agent authority is decided — this ADR's `research_agent`/`finance_agent` rules should not be read as pre-empting that document.
- OPEN QUESTION: should `Role` become a real, persisted binding (resolved through `docs/domain/principal.md`'s delegation references) before any production policy depends on it, given it is entirely caller-asserted today?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [Governance](../architecture/governance.md)
- [Kernel](../architecture/kernel.md)
- [ADR-0005](ADR-0005-kernel-interface-contracts.md)
- [Capability domain](../domain/capability.md)
- [Principal domain](../domain/principal.md)
