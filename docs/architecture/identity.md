# Identity and Authentication Architecture

Status: APPROVED

## Responsibility

Identity and Authentication establish attributable actor identity at CompanyOS trust boundaries. Identity owns durable Principal resolution, organization bindings, authentication-method trust, credential verification ports, session lifecycle, trusted claims and attestations, and normalized authenticated Principal evidence.

Identity answers **who or what presented this request, how was that established, for which organization, and with what assurance?** [Governance](governance.md) answers **may that Principal perform this Action on this Resource in this Context?** Authentication never returns `ALLOW`, `DENY`, or `REQUIRE_APPROVAL`.

No identity provider, token format, directory, key-management product, session store, or authentication protocol is selected here.

## Owns

- Principal lookup and exact external-identity/workload/provider binding resolution;
- trusted Authenticator and AttestationVerifier ports and issuer configuration;
- authentication ceremony and method classification;
- credential, key, certificate, workload, callback-signature, or token verification through adapters;
- normalized trusted claims and assurance evidence;
- authentication-session creation, binding, renewal, expiry, revocation, and replay protection;
- organization-scope binding validation;
- authentication evidence freshness, audience, nonce, issuer, and revocation checks;
- identity-related audit evidence and security events.

## Does not own

- Action authorization, policy composition, autonomy level, Authority, Approval, or separation-of-duties decisions;
- domain legality, organizational transitions, department membership semantics, or role permission semantics;
- credential issuance policy of an external provider beyond CompanyOS trust configuration;
- secrets storage, cryptographic implementation, identity database technology, network perimeter, or UI;
- model, coding-agent, provider, workflow, or workspace session semantics;
- interpreting identity attributes as permission without Governance policy.

## Authentication versus authorization

| Concern | Owner | Result |
|---|---|---|
| Authentication | Identity | `AuthenticatedPrincipalEvidence` or typed authentication failure |
| Principal lifecycle and type | [Principal domain](../domain/principal.md) | Durable Principal and organization-binding state |
| Delegated Authority and policy | Governance/Policy | Applicable Authority and policy versions |
| Authorization | Governance | `ALLOW`, `DENY`, or `REQUIRE_APPROVAL` |
| Domain legality | Kernel | Accepted decision or domain rejection |

Successful authentication is necessary but never sufficient for authorization. Authentication failure or unverifiable required claims cause Governance to fail closed; authentication success does not imply a permit.

## AuthenticatedPrincipalEvidence

The immutable evidence envelope contains:

- evidence identity, schema version, and organization ID;
- durable Principal ID/type/version and organization-binding ID/version;
- authentication session ID when interactive or session-based;
- trusted issuer and opaque issuer-subject or workload-binding reference;
- authentication method and assurance classification;
- verified ClaimAttestation identities and normalized claim names/values or protected references;
- issued, authenticated, not-before, expiry, and recorded timestamps;
- intended audience, request/channel binding, nonce or replay-protection reference;
- authenticator/verifier adapter identity and version;
- credential/key version or thumbprint reference without secret material;
- verification result, warnings, and revocation-check evidence;
- integrity protection and provenance.

The envelope is short-lived and bound to its audience and organization. It references the current durable Principal; it is not a cached replacement for loading Principal, binding, Authority, policy, or revocation state when required.

## Trusted claims and attestations

A claim is data asserted about a subject. It becomes a `TrustedClaim` only when:

1. its issuer is configured as authoritative for that claim type and organization scope;
2. its signature or equivalent integrity evidence is verified through an approved adapter;
3. subject, audience, time, nonce/channel, and organization binding match the request;
4. freshness, revocation, assurance, and claim-specific validation pass;
5. normalization preserves issuer, source value, method, and verification provenance.

A `ClaimAttestation` records issuer, subject binding, claim type, value or protected reference, issuance/expiry, evidence digest, verification method, assurance, and verifier result. Claims from agents, prompts, request bodies, provider metadata, unverified headers, or stale sessions remain untrusted context.

Trusted claims are facts for policy evaluation, not permissions. For example, a verified department-role claim may help resolve membership, but Governance still evaluates Authority and policy.

## Authentication flows by Principal type

### Human

An external identity ceremony authenticates an issuer subject; Identity resolves an active HumanPrincipal binding and organization binding, verifies required assurance and session policy, then issues evidence. A shared device or notification channel does not identify the decision maker without the required ceremony.

### Agent

An AgentPrincipal authenticates through a CompanyOS-issued workload/session binding tied to its agent identity, objective/workflow context, runtime attempt, and accountable organization. Model identity, prompt contents, conversation ID, or human request origin cannot authenticate the agent.

### Service

A ServicePrincipal authenticates using a workload identity or scoped credential bound to deployment/workload evidence and audience. Process location, network origin, environment variables, or running inside the Daemon are not sufficient identity evidence.

### Provider

A ProviderPrincipal authenticates callbacks or provider-initiated requests through verified signatures, mutually authenticated channels, or another configured method. The evidence binds provider, installation/account, endpoint, event/delivery identity, organization, and replay protection. Payload validity and authorization remain separate checks.

## Session identity versus Principal identity

An `AuthenticationSession` is a temporary authenticated interaction bound to exactly one durable Principal, one organization binding, method/assurance evidence, client or workload context, issuance/expiry, and revocation state.

- A Principal may have many sessions; a session resolves to one Principal.
- Session renewal cannot change Principal type, organization, or Authority silently.
- Provider run, model conversation, coding-agent session, workspace lease, browser session, and Runtime attempt IDs are correlation identities, not AuthenticationSessions unless explicitly bound through this contract.
- Session attributes may contribute trusted Context only when attested and current.
- Expired or revoked sessions cannot produce new authenticated evidence; historical actions retain Principal and session attribution.
- Long-running workflows reload current Principal/binding/Authority evidence before governed dispatch rather than trusting the initiating session indefinitely.

## Organization scoping

Authentication always resolves a requested organization explicitly and verifies an active `PrincipalOrganizationBinding`. Ambiguous or omitted organization scope fails. Evidence for one organization cannot be replayed in another, even when the same external identity or durable Principal participates in both.

Provider callbacks and service-to-service calls bind organization through a trusted installation/resource mapping, not an unverified payload field. Cross-organization actions require an explicit CompanyOS contract, separate evidence for each scope, and Governance evaluation.

## Delegation and revocation

Identity verifies the identities, sessions, and organization bindings involved in delegation. Governance/Policy owns creation and interpretation of delegated Authority. The authenticated delegate remains the requesting Principal; audit records retain both delegate and delegator/Authority references.

Before producing or accepting evidence, Identity checks applicable credential, session, binding, and Principal revocation state. Governance additionally checks current Authority, policy, Approval, and action-specific revocation. Cached evidence cannot outlive its expiry or override a newer revocation.

Emergency revocation emits a persisted security event and blocks new evidence. Propagation and stale-cache behavior are explicit, measurable, and fail closed for governed actions.

## Failure semantics

Normalized authentication failures include unknown binding, ambiguous Principal, inactive Principal, inactive organization binding, invalid credential, untrusted issuer, invalid attestation, insufficient assurance, wrong audience, wrong organization, not-yet-valid, expired, revoked, replayed, channel-binding mismatch, verifier unavailable, and indeterminate verification.

Unknown or indeterminate verification fails closed. Retries preserve request/correlation identity and do not bypass replay controls. Authentication errors never fall back to a weaker method unless policy explicitly permits that method for the request class.

## Governance consumption

The Application layer obtains `AuthenticatedPrincipalEvidence` from Identity and supplies its identity plus the normalized Governance request. Governance validates evidence integrity, audience, organization, freshness, Principal/binding versions, assurance requirements, and revocation status before evaluating Authority and policy.

Governance consumes the evidence; it does not authenticate credentials, map arbitrary external subjects, trust raw claims, or maintain authentication sessions. Dispatch-time re-evaluation obtains fresh evidence when the prior envelope or required session assurance is no longer valid.

## Invariants

- Every governed request identifies one durable Principal and one active organization binding through authenticated evidence.
- Authentication and authorization are separate decisions with separate owners and evidence.
- Human, agent, service, and provider types are explicit and cannot be changed by claims or sessions.
- Raw credentials, secrets, and reusable tokens never enter Principal, Governance, domain command, event, or audit payloads.
- Trusted claims retain issuer, method, assurance, time, audience, organization, verification, and provenance.
- Session identity never replaces durable Principal identity or Authority.
- Revoked, expired, ambiguous, cross-organization, replayed, or unverifiable evidence fails closed.
- Delegation narrows Authority and never changes who performed the action.
- Authentication adapters are replaceable and cannot define CompanyOS Principal or authorization semantics.
- Identity evidence and lifecycle changes are persisted and auditable before dependent governed execution.

## OPEN QUESTIONS

- Which authenticators and assurance levels are required for the first human and service flows?
- What maximum authentication-evidence and session lifetimes apply by action risk?
- Which claim issuers are authoritative for organization membership, roles, and provider installations?
- How quickly must credential, session, binding, Principal, and Authority revocation propagate?
- Which provider callback methods meet replay and organization-binding requirements initially?

## Dependencies

- [Principal domain](../domain/principal.md)
- [Events](events.md)
- [Persistence](persistence.md)
- [Security documentation](../security/README.md)
