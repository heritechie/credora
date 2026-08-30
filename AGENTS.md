# AGENTS.md

## Project

Credora is an open-source credit decisioning infrastructure project.

Positioning:

> Open-source credit decisioning infrastructure.

Principle:

> Your providers. Your keys. Your decisions.

Credora is an infrastructure/orchestration layer for building and executing credit assessment workflows using providers and data sources chosen by the customer.

Credora is NOT:

- a lender
- a credit bureau
- a KYC provider
- a fraud provider
- a proprietary scoring provider
- a provider marketplace
- a loan origination system
- a CRM
- a generic workflow automation platform
- an AI platform

---

## Core Product Loop

Everything in the core engine should support this loop:

```text
Define assessment
→ execute workflow
→ collect provider/data results
→ evaluate knockout conditions
→ calculate/evaluate credit score
→ evaluate policy/rules
→ produce decision
→ collect evidence
→ record audit trail
```

The primary product outcome is a reliable credit decision with explainable evidence.

Anything outside this loop requires explicit justification.

---

## v0 Objective

Credora v0 succeeds if a developer can:

1. Run Credora locally.
2. Define an assessment.
3. Configure mock providers.
4. Execute an assessment.
5. Handle provider failures and retries.
6. Evaluate deterministic credit rules.
7. Evaluate knockout conditions.
8. Produce:
   - APPROVE
   - REVIEW
   - REJECT
9. Explain why the decision was produced.
10. Inspect evidence and audit history.

Do not optimize v0 for enterprise administration, SaaS operations, or monetization.

Everything else is secondary.

---

## Product Direction

Credora should eventually become a playground for:

- credit analysts
- risk analysts
- underwriting teams
- risk engineering teams

Users should be able to experiment with and evaluate credit decision policies using their own rules and parameters.

The engine must therefore avoid hard-coding a single company's credit policy.

Credit rules should be configurable and deterministic.

Do not assume that one company's credit policy represents the universal definition of credit decisioning.

---

## Domain Concepts

The core domain includes:

- Assessment
- Applicant
- Application
- Provider
- Provider Result
- Credit Score
- Rule
- Policy
- Policy Version
- Knockout
- Decision
- Decision Reason
- Evidence
- Audit Event
- Backtest

These concepts should remain explicit in the domain model.

Avoid collapsing everything into a generic "workflow" abstraction.

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
- income providers
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

Use mock providers during early development.

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

## Credit Score

Credit Score is a domain concept representing a quantitative assessment used by a credit policy.

Credora should NOT initially implement a proprietary machine-learning credit scoring model.

Credora should support:

- externally calculated scores
- customer-defined score calculations
- deterministic score rules
- provider-derived scores

The exact scoring methodology must be configurable and documented.

Do not invent a universal credit-score formula.

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

## DSR

DSR (Debt Service Ratio) is a domain concept representing the relationship between debt-service obligations and the income measure used by a credit policy.

Credora must document:

- definition
- numerator
- denominator
- treatment of monthly obligations
- treatment of income
- unit/period assumptions
- edge cases
- interpretation

Do not silently assume that every organization calculates DSR identically.

Different policies may define the underlying income and debt components differently.

The implementation must keep these assumptions explicit.

---

## Knockout

Knockout is a decisioning mechanism where a condition causes an application to become ineligible or otherwise prevents normal progression through the policy.

Knockout conditions must have:

- stable rule/code
- description
- condition
- reason code
- evidence
- deterministic outcome

Do not copy proprietary knockout rules from any employer or external organization.

Use generic examples only.

A knockout should remain distinguishable from an ordinary scoring/ranking rule.

---

## Rules

Rules are deterministic policy conditions used to evaluate an assessment.

A rule should have concepts such as:

- stable ID/code
- human-readable description
- condition
- parameters
- outcome
- reason code
- evidence/reference

Do NOT introduce a complicated custom DSL unless there is a demonstrated need.

Prefer a small, strongly typed representation initially.

Rules must be:

- deterministic
- testable
- explainable
- versionable

---

## Policy

A Policy represents a versioned collection of decisioning logic.

A policy should define:

- policy ID
- version
- rules
- knockout conditions
- score configuration
- decision behavior

Policy evaluation must be reproducible.

A historical assessment must be explainable using the policy version that produced its decision.

Never silently evaluate historical data using the current policy version.

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

## Decisioning

Decisioning must be deterministic and explainable.

Supported initial decisions:

- APPROVE
- REVIEW
- REJECT

Every final decision should have structured reasons.

Conceptual example:

```json
{
  "decision": "REJECT",
  "reasons": [
    {
      "code": "HIGH_DSR",
      "description": "Debt service ratio exceeds policy threshold",
      "value": 0.81,
      "threshold": 0.70
    }
  ],
  "policy": {
    "id": "personal-loan",
    "version": 3
  }
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

Avoid returning only a boolean or opaque score without explanation.

---

## Evidence

Evidence is first-class.

Evidence should allow the system to explain:

- where a value came from
- which provider produced it
- when it was retrieved
- which rule used it
- which decision reason references it

Provider outputs and important decision inputs should be captured as evidence.

Conceptual example:

```json
{
  "source": "credit_provider",
  "field": "monthly_obligation",
  "value": 3500000,
  "retrieved_at": "...",
  "reference": "provider-result-id"
}
```

Do not store unexplained decision outputs without their supporting evidence.

Use structured data.

PostgreSQL JSONB is acceptable for provider/evidence payloads.

Do not store unnecessary PII.

---

## Backtesting

Backtesting is a planned core capability.

The goal is to allow credit/risk users to test a decision policy against historical/real production data.

Example:

```text
Historical applications
→ Policy v1
→ Policy v2
→ Policy v3
→ compare decisions and outcomes
```

Backtesting should eventually answer questions such as:

- How many applications would be approved?
- How many would be rejected?
- Which applications changed decision?
- Which rules caused the changes?
- What would approval/rejection rates look like under another policy?
- How does a new policy compare to a previous policy?

IMPORTANT:

Backtesting must use the same deterministic policy evaluation engine as live decisioning.

Do not create a separate rule implementation for backtesting.

Backtesting is NOT part of the first implementation unless explicitly requested.

Design the engine so policy evaluation can later accept both:

- live assessment data
- historical assessment data

Do not put live-only assumptions inside the policy evaluator.

---

## Explainability

Every decision should be explainable.

Prefer structured reasons:

```json
{
  "code": "...",
  "description": "...",
  "value": "...",
  "threshold": "...",
  "evidence": "..."
}
```

Avoid opaque responses such as:

```json
{
  "score": 712,
  "decision": "REJECT"
}
```

without explanation.

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

Treat the following as sensitive:

- API credentials
- PII
- financial data
- credit data
- identity data
- provider responses

Never:

- hard-code secrets
- commit credentials
- log API keys
- unnecessarily log raw PII
- expose provider credentials to clients

Security-sensitive decisions must be explicit.

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

Credit decisioning logic requires strong deterministic tests.

Required coverage includes:

### Unit tests

- rule evaluation
- knockout evaluation
- score calculation
- decision aggregation
- policy versioning
- explainability
- evidence references
- deterministic repeated execution
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
→ knockout
→ score
→ policy
→ decision
→ evidence
→ audit
```

Prefer table-driven tests in Go where appropriate.

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

## Documentation

Domain documentation is important.

Document explicitly:

- DSR
- Credit Score
- Rule
- Policy
- Policy Version
- Knockout
- Decision
- Evidence
- Backtest

Definitions must be Credora's documented domain definitions.

Do not present assumptions from a particular employer or provider as universal industry truth.

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
2. Does a developer, credit analyst, or risk team need it?
3. Does it improve explainability or reproducibility?
4. Does it improve provider abstraction?
5. Is the complexity justified?
6. Can a simpler implementation achieve the same result?

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
- CRM
- LOS
- generic AI platform
- proprietary ML scoring platform
- cloud management platform

unless explicitly approved.

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

## Development Workflow

Before implementing a feature:

1. Inspect the existing code.
2. Understand existing domain boundaries.
3. Prefer the smallest viable implementation.
4. Write tests for domain behavior.
5. Implement.
6. Run formatting.
7. Run tests.
8. Run static analysis where applicable.
9. Review the diff.
10. Report what changed and what was intentionally deferred.

Do not commit or push changes unless explicitly requested.

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

## Current Phase

The immediate focus is:

Credora v0 domain and decision engine foundation.

Priorities:

1. domain model
2. deterministic rule evaluation
3. knockout evaluation
4. score handling
5. policy/versioning
6. decision aggregation
7. evidence
8. audit trail
9. mock providers
10. reliable assessment execution

Backtesting is a planned capability and should influence the architecture, but should not cause premature implementation complexity.

Do not build a UI or SaaS layer before the core engine is credible.

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
