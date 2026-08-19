# Principal Domain

Status: DRAFT

## Definition

A `Principal` is the durable CompanyOS identity of a human, agent, service, or provider that may participate in an organization. It answers **who or what is acting**, not what it is permitted to do.

A Principal is not a credential, login, API key, session, model conversation, provider run, department role, membership, Authority, Approval, or policy decision. Those records may reference a Principal but cannot replace its stable identity.

## Principal types

### HumanPrincipal

A durable identity for a natural person. It records the minimum organizational identity attributes and verified external-identity bindings required for attribution. Email address, username, employee number, or identity-provider subject may be an alias or binding, but none is the Principal ID.

Human identity does not imply ownership, employment, department membership, approver eligibility, or authority. Those are separately governed relationships.

### AgentPrincipal

A durable identity for a CompanyOS-managed computational participant. It records creating organization, agent definition/profile reference, accountable owner, lifecycle, and provenance. A new model response, conversation, run, replica, or resumed session does not create a new AgentPrincipal unless a governed domain operation explicitly does so.

An AgentPrincipal may receive delegated Authority but can never approve its own action or become human merely because a human initiated its session.

### ServicePrincipal

A durable identity for an internal workload, daemon component, automation, integration adapter, or deployed service. Replicas and restarted processes normally authenticate as the same ServicePrincipal while using distinct sessions and workload attestations.

A ServicePrincipal has no authority merely because it runs inside CompanyOS or shares a process with an authorized component.

### ProviderPrincipal

A durable identity for an external provider organization, installation, account, endpoint, or provider-controlled service acting across a CompanyOS trust boundary. It is distinct from a provider profile, model, coding-agent profile, adapter, API credential, callback delivery, and provider session.

ProviderPrincipal evidence attributes an external action or callback. It does not make provider claims trusted, grant access to organizational state, or authorize the provider to initiate arbitrary CompanyOS commands.

## Principal record

Every Principal contains:

- globally unique stable Principal ID and Principal type;
- lifecycle status and version;
- canonical display reference with minimal identifying attributes;
- creation method, creating Principal, organization, and provenance;
- active organization bindings and their versions;
- external identity or workload bindings by opaque issuer/subject references;
- accountable human/organizational owner where required for non-human Principals;
- security classification and audit/retention references;
- suspension, revocation, retirement, and supersession evidence;
- created, effective, expiry where applicable, and recorded timestamps.

Secrets, reusable credentials, raw identity tokens, biometric material, and unrestricted provider payloads are not stored in the Principal record.

## Organization binding

A `PrincipalOrganizationBinding` relates one durable Principal to one organization with a stable binding ID, status, effective interval, identity-assurance requirements, permitted authentication methods, accountable issuer, and membership/role references when applicable.

Every organizational request names exactly one organization and one active binding. A Principal present in multiple organizations uses a distinct binding and independently evaluated Authority in each. No alias, session, credential, department membership, or provider account silently crosses an organization boundary.

Organization binding establishes identity participation only. It grants no Action permission.

## Lifecycle

Proposed Principal lifecycle:

```text
PROPOSED -> ACTIVE -> SUSPENDED -> ACTIVE
    |          |          \------> REVOKED
    |          \-----------------> RETIRED
    \----------------------------> REJECTED
```

- `ACTIVE` permits authentication subject to an active organization binding and valid method.
- `SUSPENDED` blocks new authenticated evidence while preserving identity and audit history.
- `REVOKED` permanently invalidates the Principal for new actions; recovery requires a new Principal unless a future approved rule permits reversal.
- `RETIRED` ends expected use without erasing historical attribution.

Lifecycle transitions are governed, versioned, persisted, and attributable. Deleting an external account or credential does not delete the Principal or its historical actions.

## Delegation

Delegation creates or changes a versioned `Authority` relationship from one Principal to another; it does not mutate Principal identity. The delegation references delegator, delegate, organization binding, allowed Actions/Resources/context, validity, depth, constraints, issuer, and revocation evidence according to the [Policy domain](policy.md).

Delegation can only narrow authority. A Principal cannot delegate more than its own current delegable Authority, delegate across organizations without an explicit governed relationship, or use a new session to escape delegation limits.

## Revocation distinctions

- **Credential revocation** invalidates one authentication mechanism.
- **Session revocation** terminates one or more authentication sessions.
- **Binding revocation** prevents the Principal acting in one organization.
- **Authority revocation** removes a delegation or permission scope.
- **Principal revocation** prevents the durable identity from initiating new actions everywhere it applies.

These are separate state changes with separate evidence. Governance evaluates the current combination; no revocation is inferred merely from another unless an explicit policy defines cascading behavior.

## Invariants

- Every actor is represented by one durable Principal identity before it can request a governed action.
- Principal type is explicit and cannot change through session, role, prompt, or provider claims.
- Principal identity grants no Authority, Approval, membership, or Action permission by itself.
- Every organizational action binds the Principal to exactly one active organization context.
- External aliases and provider subjects map to Principals through verified versioned bindings, never string equality alone.
- Agent, service, and provider Principals identify accountable non-human actors without pretending they are humans.
- Sessions and credentials are replaceable authentication mechanisms, not durable identities.
- Suspension, revocation, retirement, alias changes, and binding history never erase prior attribution.
- Delegation narrows authority and remains separate from identity lifecycle.
- Principal and organization-binding changes are governed, persisted, version-checked, and auditable.

## OPEN QUESTIONS

- Are human Principals global across organizations or organization-local with explicit federation links?
- Which non-human Principal types require an accountable HumanPrincipal at all times?
- Can a revoked Principal ever be restored, or must restoration create a new identity?
- Which identity attributes may be retained or displayed under privacy and erasure requirements?
- How are provider organizations, installations, and individual provider workloads related?

## Dependencies

- [Policy](policy.md)
- [Approval](approval.md)
- Future Organization, Membership, Role, Agent, and Provider domain definitions
