# Approval Domain

Status: DRAFT

## Definition

An Approval is durable human authorization evidence for one bounded request that Governance classified as `REQUIRE_APPROVAL`. It does not execute the Action, create Authority, override a forbid, or make an illegal workflow transition legal.

## Approval record

An Approval records:

- stable approval ID and organization scope;
- requesting Principal and request/decision correlation IDs;
- Action and Resource identities;
- canonical proposed-arguments digest and governed reference to sensitive details;
- objective, workflow, and execution context needed to understand the request;
- policy, authority, resource, and request versions evaluated;
- required approver constraints and approval count;
- status, expiry, allowed uses, and consumption state;
- decision makers, authenticated channels, timestamps, comments, and reasons;
- supersession, revocation, and execution-evidence correlations.

## Lifecycle

Proposed lifecycle:

```text
PENDING -> APPROVED -> CONSUMED
        -> REJECTED
        -> EXPIRED
        -> CANCELLED

APPROVED -> REVOKED
```

- `PENDING`: persisted and awaiting eligible human decision makers.
- `APPROVED`: required decisions are satisfied, but execution has not been authorized or performed.
- `CONSUMED`: an `ALLOW` decision used this Approval for its bounded request.
- `REJECTED`: an eligible decision maker refused the request.
- `EXPIRED`: the decision window closed without usable approval.
- `CANCELLED`: the requester or workflow withdrew the request before resolution.
- `REVOKED`: previously approved evidence was withdrawn before valid consumption.

Terminal transitions are immutable. Reconsideration creates a new Approval rather than rewriting history.

## Resolution and execution

1. Governance evaluates the immutable proposal and produces an attributable `REQUIRE_APPROVAL` Decision with the applicable approval constraints; it does not persist the result directly.
2. Application atomically persists the Decision, immutable `PendingCommand`, `PENDING` Approval, audit record, and idempotency result before returning `APPROVAL_REQUIRED` or notifying Runtime. This transaction creates no target-domain transition or execution intent.
3. An authenticated human decision maker reviews the immutable request details and approves or rejects.
4. Application coordinates resolution through an atomic status/version transition with its audit and idempotency records so duplicate or competing decisions cannot overwrite one another.
5. Runtime wakes only after the resolved state is persisted; notification delivery alone is not resolution.
6. On resume, Application reloads the pending command and authoritative state, and Governance re-evaluates policy, Authority, context, request digest, expiry, revocation, and approver eligibility.
7. A matching Approval may produce a fresh `ALLOW`; Application must still obtain the final Kernel decision before atomically persisting the target-domain transition, events, Approval consumption, pending-command closure, and execution intent.

## Invariants

- Only authenticated, authorized human Principals can approve or reject.
- Agents and services cannot approve, simulate approval, or select an approver to bypass policy.
- The requester cannot approve its own request unless an explicit future human separation-of-duties policy permits it; agent self-approval is always prohibited.
- Approval never overrides `DENY`, missing Authority, emergency stop, organization isolation, or Kernel illegality.
- Any material change to Principal, Action, Resource, arguments, objective/workflow context, policy, or Authority requires re-evaluation and normally a new Approval.
- Approval is explicit; timeout and silence never mean approval.
- Resolution is atomic, durable, attributable, and append-only in audit history.
- Approved is not executed; execution failure does not rewrite the approval decision.
- Single-use is the safe default until bounded reusable grants are explicitly specified.
- Notification channels carry requests and decisions but do not own Approval state.
- Sensitive inputs are minimized and protected while remaining reviewable through governed references.

## Audit requirements

Audit evidence must distinguish request creation, notification delivery, view/access where required, each human decision, expiry/cancellation/revocation, governance re-evaluation, consumption, dispatch, and execution result. Every record identifies actor, channel, timestamp, prior version, resulting version, and correlation IDs.

## Relationship to policy

[Policy](policy.md) determines whether approval is required and who may decide. [Governance](../architecture/governance.md) owns evaluation and verifies Approval evidence. [Application](../architecture/application.md) coordinates atomic persistence. Runtime owns durable waiting and resume, not Approval meaning or persistence coordination.

## Open questions

- OPEN QUESTION: Which action classes require one approver, multiple approvers, or role separation?
- OPEN QUESTION: What are the default expiry and single-use rules by risk class?
- OPEN QUESTION: May an approver edit a request, or must every edit create a new request and Approval?
- OPEN QUESTION: Which notification and decision channels satisfy identity assurance requirements?
- OPEN QUESTION: When, if ever, can a bounded reusable human grant be represented as Authority instead of Approval?

## Dependencies

- [Organization](organization.md)
- [Principal](principal.md)
- [Policy](policy.md)
