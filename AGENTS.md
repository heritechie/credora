# AGENTS.md

## Project

Credora is an open-source credit decisioning infrastructure project.

Positioning:

> Open-source credit decisioning infrastructure.

Principle:

> Your providers. Your keys. Your decisions.

Credora is an infrastructure/orchestration layer for building and executing credit assessment workflows using providers and data sources controlled by the customer.

Credora is NOT:

- a lender
- a credit bureau
- a KYC provider
- a fraud provider
- a proprietary scoring provider
- a provider marketplace
- a loan origination system
- a generic workflow automation platform

---

## Core Product Loop

Everything in the core engine should support this loop:

```text
Define assessment
    ↓
Execute workflow
    ↓
Collect provider data
    ↓
Evaluate policy
    ↓
Produce decision
    ↓
Collect evidence
    ↓
Record audit trail
```

The primary product outcome is a reliable credit decision with explainable evidence.

---

## Engineering Philosophy

Prioritize:

1. Correctness
2. Simplicity
3. Explicit domain models
4. Maintainability
5. Testability
6. Security
7. Developer experience
8. Extensibility

Prefer the simplest implementation that satisfies the requirement.

Do not introduce abstractions merely because they may be useful in the future.

Do not build infrastructure before the product requires it.

Avoid speculative architecture.

---

## Architecture

Use a modular monolith initially.

The preferred architecture is:

```text
HTTP API
   ↓
Application Layer
   ↓
Domain
   ↓
Ports / Interfaces
   ↓
Infrastructure
```

The domain must not depend on:

- HTTP
- PostgreSQL
- Temporal
- Docker
- specific providers

Infrastructure implementations may depend on the domain interfaces.

---

## Technology

Preferred stack:

- Go for the core engine
- PostgreSQL
- REST/HTTP
- Docker
- Docker Compose

Use standard Go tooling whenever practical.

Do not introduce additional infrastructure unless there is a concrete requirement.

Avoid premature adoption of:

- Kafka
- RabbitMQ
- Redis
- Kubernetes
- GraphQL
- gRPC
- service mesh
- CQRS
- event sourcing

---

## Temporal

Temporal is an implementation detail, not the product.

Do not introduce Temporal merely because Credora contains workflows.

Use Temporal when the system actually requires:

- durable execution
- retries
- timeouts
- failure recovery
- asynchronous execution
- long-running workflows
- workflow state persistence

Keep the execution boundary abstract enough that Temporal can be introduced later.

Example:

```go
type AssessmentExecutor interface {
    Execute(ctx context.Context, assessmentID string) error
}
```

The first implementation may be synchronous.

Do not make the domain depend directly on Temporal APIs.

---

## Provider Architecture

External providers must always be replaceable.

Examples include:

- identity providers
- KYC providers
- credit bureaus
- fraud providers
- scoring services
- other external data providers

The domain depends on interfaces.

Example:

```go
type CreditProvider interface {
    GetCreditData(
        ctx context.Context,
        input CreditInput,
    ) (CreditResult, error)
}
```

Provider implementations must not contain credit policy logic.

Provider adapters are responsible for:

- authentication
- request construction
- provider-specific response mapping
- provider-specific error handling
- normalization into Credora domain types

The rest of Credora should not depend on provider-specific response formats.

---

## BYOK

Credora follows:

> Your providers. Your keys. Your decisions.

Customers own external provider credentials.

Never:

- hard-code API keys
- commit secrets
- log credentials
- store plaintext credentials
- make the architecture dependent on one provider

Secret management should be introduced carefully.

Do not build a complex credential-management platform in the MVP.

---

## Assessment

An assessment represents one execution of a credit decisioning process.

At minimum it should have:

```text
id
application_id
policy_id
status
created_at
started_at
completed_at
```

Statuses should distinguish execution state from business decision.

Example:

```text
PENDING
RUNNING
COMPLETED
FAILED
```

Business decisions:

```text
APPROVE
REVIEW
REJECT
```

Never represent a technical execution failure as `REJECT`.

---

## Decisioning

Decisioning must be deterministic and explainable.

Every final decision should have structured reasons.

Example:

```json
{
  "decision": "REJECT",
  "reasons": [
    {
      "code": "HIGH_DSR",
      "value": 0.81,
      "threshold": 0.7
    }
  ],
  "policy": "personal-loan-v3"
}
```

Reason codes should be stable identifiers.

Examples:

```text
HIGH_DSR
LOW_CREDIT_SCORE
HIGH_FRAUD_SCORE
IDENTITY_FAILED
PROVIDER_ERROR
```

Avoid returning only free-form human-readable explanations.

---

## Policy

Policies are deterministic.

Policy evaluation should be isolated behind an interface.

Example:

```go
type Policy interface {
    Evaluate(
        ctx context.Context,
        input PolicyInput,
    ) (PolicyResult, error)
}
```

Do not build a generic rules language unless explicitly required.

Do not introduce:

- Lua
- CEL
- JavaScript execution
- arbitrary code execution

for policy evaluation during the initial implementation.

Start with explicit Go policies.

---

## Scoring

Credora does not initially provide a proprietary machine-learning scoring model.

Scoring is one component of the decisioning pipeline.

Keep scoring behind an interface:

```go
type Scorer interface {
    Calculate(
        ctx context.Context,
        input ScoreInput,
    ) (ScoreResult, error)
}
```

Mock/deterministic scoring is sufficient for the MVP.

Customers should eventually be able to bring their own scoring model or service.

---

## Evidence

Evidence is first-class.

Provider outputs and important decision inputs should be captured as evidence.

Examples:

```text
IDENTITY_RESULT
CREDIT_REPORT
FRAUD_RESULT
SCORE_RESULT
POLICY_RESULT
```

Evidence should allow a developer/operator to understand how a decision was produced.

Use structured data.

PostgreSQL JSONB is acceptable for provider/evidence payloads.

Do not store unnecessary PII.

---

## Audit Trail

Important assessment events must be auditable.

Examples:

```text
ASSESSMENT_CREATED
ASSESSMENT_STARTED
PROVIDER_EXECUTED
PROVIDER_FAILED
POLICY_EVALUATED
DECISION_PRODUCED
ASSESSMENT_COMPLETED
ASSESSMENT_FAILED
```

Audit records should be append-oriented.

Do not implement full event sourcing.

Do not use the audit table as the primary application state.

---

## Error Handling

Always distinguish:

### Business outcome

```text
APPROVE
REVIEW
REJECT
```

from:

### Technical execution state

```text
COMPLETED
FAILED
```

Provider/network/database failures must not silently become business rejection decisions.

Errors should be:

- explicit
- structured
- observable
- testable

---

## Idempotency

Assessment execution must be designed with retry safety in mind.

Provider calls should be safe to retry where possible.

Avoid creating duplicate final decisions or contradictory assessment states.

Do not build a distributed idempotency platform unless required.

Prefer simple deterministic behavior.

---

## Security

Treat credit data as sensitive.

Never log:

- API keys
- passwords
- tokens
- national ID numbers
- unnecessary PII
- complete provider credentials

Be conservative with logs.

Configuration must come from environment/configuration rather than source code.

Use `.env.example`, never commit `.env`.

---

## Database

PostgreSQL is the default persistence layer.

Use migrations.

Initial core tables:

```text
assessments
evidence
audit_events
```

Do not over-normalize provider payloads.

JSONB is acceptable for:

- provider data
- evidence
- audit payloads
- decision reasons

Database access must remain outside the domain layer.

---

## API

REST/HTTP initially.

Keep HTTP handlers thin.

Handlers should:

1. Validate request
2. Call application service
3. Map domain result to HTTP response

Handlers must not contain:

- provider logic
- policy logic
- scoring logic
- database queries
- workflow orchestration

---

## Testing

Tests are part of the implementation, not a later step.

Required coverage includes:

### Unit tests

- policy evaluation
- score calculation
- decision reasons
- assessment orchestration
- provider failures

### Integration tests

- assessment creation
- assessment execution
- assessment retrieval
- evidence retrieval
- audit retrieval
- provider failure behavior

The happy path must verify:

```text
create
→ execute
→ identity
→ credit
→ fraud
→ score
→ policy
→ decision
→ evidence
→ audit
```

---

## Observability

Use structured logging.

Important events should be observable:

```text
assessment.created
assessment.started
provider.completed
provider.failed
policy.evaluated
decision.produced
assessment.completed
assessment.failed
```

Do not add a complex observability stack during MVP.

---

## Repository Structure

Preferred structure:

```text
apps/
    landing/

services/
    engine/
        cmd/
        internal/
            assessment/
            provider/
            policy/
            decision/
            evidence/
            audit/
        migrations/

examples/

deploy/

docs/
```

The core engine should remain independent from the landing page.

---

## Landing Page

The landing page is an Astro application.

Its purpose is to communicate:

- what Credora is
- why it exists
- architecture
- developer experience
- open-source positioning

Do not allow marketing/UI work to dominate engine development.

The landing page must not dictate domain architecture.

---

## Scope Guard

Before implementing a feature, ask:

1. Does it improve the core credit decisioning loop?
2. Does a developer need it to execute an assessment?
3. Does it improve reliability?
4. Does it improve explainability or evidence?
5. Does it improve provider abstraction?
6. Does it materially improve OSS developer experience?

If the answer is mostly "no", defer the feature.

Do not implement speculative enterprise functionality.

Explicitly defer:

- billing
- dashboards
- multi-tenancy
- complex RBAC
- provider marketplace
- dozens of provider integrations
- ML platform
- cloud hosting
- Kubernetes
- enterprise administration
- notification systems
- analytics platform
- generic workflow builder

---

## Change Discipline

Before making architectural changes:

1. Inspect the existing implementation.
2. Understand existing interfaces.
3. Prefer modifying the smallest number of components.
4. Avoid unnecessary refactors.
5. Preserve existing tests.
6. Add tests for behavior being changed.

Do not rewrite working code merely for stylistic preference.

Do not introduce a new dependency when the standard library or an existing dependency is sufficient.

---

## Dependency Discipline

Before adding a dependency, evaluate:

- Is it actually necessary?
- Can the standard library solve the problem?
- Is it maintained?
- Does it significantly increase the maintenance burden?
- Does it introduce security risk?
- Is it justified for an open-source project?

Prefer fewer dependencies.

---

## Developer Experience

A developer should eventually be able to:

```bash
git clone ...
docker compose up
```

and run a complete assessment locally.

Documentation and examples are part of the product.

Prefer:

- clear commands
- reproducible setup
- useful errors
- deterministic examples
- straightforward configuration

---

## Current MVP Definition

Credora v0.1 succeeds when a developer can:

1. Run Credora locally.
2. Define an assessment.
3. Configure provider implementations.
4. Execute an assessment.
5. Handle provider failures.
6. Apply deterministic policies.
7. Produce APPROVE / REVIEW / REJECT.
8. Explain the decision.
9. Inspect evidence.
10. Inspect the audit trail.

Everything else is secondary.

---

## Agent Behavior

When working on Credora:

- Be skeptical of scope expansion.
- Prefer concrete implementation over speculative abstraction.
- Identify architectural risks before coding.
- Do not silently add features.
- Do not assume enterprise requirements.
- Do not introduce infrastructure without a requirement.
- Keep provider integrations replaceable.
- Keep business rules deterministic.
- Keep decisions explainable.
- Keep secrets out of source control.
- Keep the core engine small.

If a requested change conflicts with this document, explain the conflict before implementing it.

When uncertain, prefer the smaller implementation.
