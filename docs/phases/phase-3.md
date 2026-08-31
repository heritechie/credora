# Phase 3: Decisioning Workspace

## Objective

Phase 3 introduces the **Credora Decisioning Workspace** — a developer and analyst-facing interface for interacting with Credora's credit decisioning capabilities.

The Workspace is not a generic dashboard. It is a focused tool for understanding, inspecting, and testing credit policy evaluation. It exists to make the decisioning engine accessible to the humans who build, operate, and reason about credit policies.

---

## Target Personas

### Developer

Developers use the Workspace to:

- Inspect assessments created via the API
- Test API behavior interactively
- Inspect policy evaluation execution
- Inspect decision reasons and evidence
- Debug decisioning behavior during development
- Verify that policy changes produce expected outcomes

Developers need precision, raw data access, and the ability to trace exactly what happened during evaluation.

### Credit Analyst

Credit analysts use the Workspace to:

- Understand what policies exist and what they do
- Run assessment simulations with specific applicant/application data
- Inspect why an assessment was approved, reviewed, or rejected
- Understand which rules, knockouts, and score thresholds fired
- Inspect the decision outputs (credit limit, approved amount)
- Compare how the same assessment behaves under different policy versions

Analysts need clarity, structured explanation, and the ability to reason about policy behavior without reading Go source code.

### Risk Team

Risk teams use the Workspace to:

- Review policy behavior before deployment
- Compare policy versions side-by-side
- Inspect decision impact of policy changes
- Understand which rules cause rejections or reviews
- Eventually support backtesting and portfolio-level analysis

Risk teams need reproducibility, comparison, and audit-quality explanation.

**Note:** Full backtesting and portfolio analytics are future capabilities, not Phase 3 MVP requirements.

---

## Core MVP Capabilities

The Phase 3 MVP is defined by four capabilities:

### A. Policy View

The Workspace presents available policies and their versions. A user can:

- See which policies are registered
- See the versions available for each policy
- See policy metadata (name, description)
- Understand what a policy evaluates (knockouts, rules, score thresholds)

This is a read-only view. The Workspace does not edit policies in Phase 3.

### B. Assessment Simulator

The Workspace provides an interface for creating and running an assessment. The simulator must support the current domain semantics:

**Scenario A: Limit Assessment (no application)**

```text
Applicant
  ↓
Policy
  ↓
Assessment
  ↓
Credit Limit
```

- Applicant is provided
- No application
- Policy determines credit limit
- Result: credit limit output

**Scenario B: Loan Application (with requested amount)**

```text
Applicant
Application (with requested amount)
  ↓
Policy
  ↓
Assessment
  ↓
Decision: APPROVE / REVIEW / REJECT
```

- Applicant and application provided
- Requested amount specified
- Policy evaluates knockouts, rules, score thresholds
- Result: decision with reasons and outputs

**Scenario C: Application without requested amount**

```text
Applicant
Application (without requested amount)
  ↓
Policy
  ↓
Assessment
  ↓
Decision
```

- Application exists but no specific amount requested
- Policy evaluates eligibility
- Result: decision

**Important:** Not every credit assessment is tied to a requested loan amount. The simulator must not require a requested amount.

### C. Decision Explanation

After running an assessment, the Workspace presents the decision in a structured, explainable format:

**Decision View:**

```text
Decision:     APPROVE
Policy:       personal-loan:v3
```

**Reasons (when triggered):**

```text
Reason:       HIGH_DSR
Description:  Debt service ratio exceeds policy threshold
Value:        0.81
Threshold:    0.70
Evidence:     HIGH_DSR
```

**Outputs (when produced):**

```text
Credit Limit:    Rp 15,000,000
Approved Amount: Rp 10,000,000
```

**Evidence:**

```text
Source:       rule
Field:        HIGH_DSR
Value:        true
Retrieved At: 2025-01-15T10:30:00Z
Reference:    HIGH_DSR
```

The Workspace must make it clear:

- Which policy version was used
- Which knockouts were evaluated (and whether they triggered)
- Which rules were evaluated (and whether they triggered)
- Which score thresholds were applied
- What evidence supported each decision reason
- What outputs were produced

### D. Policy Version Comparison

The Workspace supports deterministic single-assessment comparison against two policy versions.

**Input:** One assessment scenario, two policy versions.

**Output:**

```text
Applicant X

Policy v1:
  Decision:   APPROVE
  Credit Limit: Rp 15,000,000
  Reasons:     (none)

Policy v2:
  Decision:   REVIEW
  Credit Limit: (none)
  Reasons:     HIGH_DSR (0.81 > 0.70)
```

The comparison identifies meaningful differences:

- Outcome changed (APPROVE → REVIEW)
- Reasons changed (new or removed reasons)
- Outputs changed (credit limit present/absent, different values)
- Policy version changed

**Do NOT implement:**

- Portfolio backtesting
- Historical dataset analysis
- Statistical analysis
- Batch comparison

Those belong to a future Backtesting phase.

---

## Product Boundary

### Phase 3 IS

A decisioning workspace for:

- Running assessments interactively
- Understanding decisions
- Inspecting evidence
- Comparing policy versions

### Phase 3 is NOT

- A generic analytics dashboard
- A policy editor
- A backtesting platform
- A CRM
- A loan origination system
- A collections system
- A monitoring dashboard
- A provider management console

### Explicitly Deferred

| Capability | Status |
|---|---|
| Full policy editor | Deferred (requires structured policy model first) |
| Custom DSL | Deferred (current Go policies are sufficient for MVP) |
| Backtesting | Deferred (future phase) |
| Portfolio analytics | Deferred (future phase) |
| Production monitoring | Deferred |
| Provider marketplace | Deferred |
| Provider management | Deferred |
| BYOK credential management | Deferred |
| Authentication | Deferred |
| RBAC | Deferred |
| Multi-tenancy | Deferred |
| Billing | Deferred |
| Loan origination | Deferred |
| CRM | Deferred |
| Collections | Deferred |
| Workflow builder | Deferred |
| ML model training | Deferred |
| AI-generated policies | Deferred |
| Cloud infrastructure | Deferred |
| Arbitrary charts | Deferred |
| Vanity metrics | Deferred |
| KPI dashboards | Deferred |

If a feature does not directly support:

```text
Define assessment
→ Execute decision
→ Explain decision
→ Compare policy behavior
```

it is outside Phase 3 MVP.

---

## Policy Model — Architectural Decision

### Current Implementation

Current Credora policy evaluation uses executable Go conditions:

```go
type Rule struct {
    Code        string
    Description string
    Condition   func(Assessment) ConditionResult
    Outcome     RuleOutcome
    ReasonCode  string
    ReasonDesc  string
}

type Knockout struct {
    Code        string
    Description string
    Condition   func(Assessment) ConditionResult
    ReasonCode  string
    ReasonDesc  string
}
```

Policy definitions live in Go code (the policy registry). Policy metadata (ID, version, name, description) is persisted in the database.

### The Limitation

The current executable policy model is developer-oriented. Phase 3 needs to define how policy definitions can eventually become representable as structured data suitable for analyst-facing workflows.

However, the exact Policy Definition Model is an **unresolved architectural decision**.

### Why This Decision Matters

The policy model affects:

- **Determinism:** Structured rules must evaluate identically to Go functions
- **Explainability:** Analysts must understand what a rule does without reading Go code
- **Versioning:** Structured definitions must be versionable and reproducible
- **Reproducibility:** Historical assessments must remain explainable
- **UI representation:** The Workspace needs structured data to render policy information
- **Future backtesting:** The same evaluation engine must work for live and historical data
- **Compatibility:** Structured policies must coexist with existing Go policies

### Potential Future Shape

A structured rule might conceptually look like:

```json
{
  "code": "LOW_CREDIT_SCORE",
  "description": "Credit score below 650 is rejected",
  "condition": {
    "field": "score.value",
    "operator": "lt",
    "threshold": 650
  },
  "outcome": "reject",
  "reason_code": "LOW_CREDIT_SCORE",
  "reason_description": "Credit score is below the minimum threshold"
}
```

But this is **not** a committed design. The following questions remain open:

- What operators should be supported?
- How should nil/missing fields be handled?
- How should complex conditions (compound logic) be represented?
- How should conditions reference different input fields (applicant, application, score)?
- Should conditions be stored as JSONB in the database?
- How should the structured evaluator verify equivalence with Go functions?
- Should structured policies be a separate evaluation path or replace Go policies?

### Recommendation

Phase 3 should NOT introduce a DSL or structured policy model as part of the MVP. The MVP should:

1. Use existing Go policies as the source of truth
2. Expose policy metadata (knockout codes, rule codes, descriptions) via the API
3. Present this metadata in the Workspace for analyst understanding
4. Leave the structured policy model as an explicit follow-up decision

This avoids premature abstraction while still making policy information visible to analysts.

---

## Policy Versioning

Phase 3 must respect the existing versioning model:

```text
Policy ID + Version = Policy Identity
```

Example:

```text
personal-loan:v1
personal-loan:v2
personal-loan:v3
```

### Invariants

- Policy versions are immutable once used by an assessment
- An assessment evaluated against v1 must continue to mean v1 even after v2 exists
- The Workspace must always display which policy version produced a decision
- Creating a new policy version does not modify previous versions
- Policy comparison must use the same assessment input against two specific versions

### Workspace Behavior

- The policy selector shows all available versions
- The user explicitly chooses a version
- The chosen version is always visible in the decision view
- Policy comparison requires selecting two specific versions

---

## Assessment Simulator

### Input Model

The simulator must support the current domain input model:

```text
Applicant (required)
  ├── id: string
  ├── name: string
  └── age: int

Application (optional)
  ├── id: string
  ├── requested_amount: int64 (optional, smallest currency unit)
  └── purpose: string

CreditScore (optional)
  ├── value: int
  └── provider: string

Policy (required for selection)
  ├── id: string
  └── version: int
```

### Required Scenarios

The simulator must support at minimum:

**Limit Assessment:**

```json
{
  "applicant": { "id": "a-001", "name": "Jane Doe", "age": 35 },
  "policy": { "id": "personal-loan", "version": 1 }
}
```

**Loan Application with Score:**

```json
{
  "applicant": { "id": "a-001", "name": "Jane Doe", "age": 35 },
  "application": {
    "id": "app-001",
    "requested_amount": 10000000,
    "purpose": "working_capital"
  },
  "score": { "value": 720, "provider": "mock-credit-bureau" },
  "policy": { "id": "personal-loan", "version": 1 }
}
```

**Application without Amount:**

```json
{
  "applicant": { "id": "a-001", "name": "Jane Doe", "age": 35 },
  "application": {
    "id": "app-001",
    "purpose": "working_capital"
  },
  "policy": { "id": "personal-loan", "version": 1 }
}
```

### Behavior

- The simulator creates an assessment via the existing API
- The assessment is evaluated by the backend engine
- The Workspace displays the result
- The simulator does NOT implement evaluation logic in the frontend

---

## Decision Explanation

### Required Information

Every decision view must present:

1. **Outcome:** APPROVE, REVIEW, or REJECT
2. **Policy:** ID and version used
3. **Reasons:** Structured reasons with code, description, value, threshold, evidence ref
4. **Outputs:** Credit limit and/or approved amount (when produced)
5. **Evidence:** All evidence entries collected during evaluation

### Evaluation Trace

The Workspace should present the full evaluation trace:

**Knockouts evaluated:**

```text
AGE_BELOW_MINIMUM: NOT triggered (age=35, threshold=18)
DSR_ABOVE_MAXIMUM: TRIGGERED (dsr=0.81, threshold=0.70)
```

**Rules evaluated:**

```text
HIGH_REQUEST_AMOUNT: NOT triggered (no requested amount)
LOW_CREDIT_SCORE: NOT triggered (score=720, threshold=650)
```

**Score thresholds:**

```text
Approve threshold: 700 (score=720 → passes)
Review threshold: 500 (score=720 → passes)
```

This level of detail helps analysts and developers understand exactly what happened during evaluation.

---

## Policy Comparison

### Comparison Model

```text
Assessment Input
    ↓
 ┌──┴────────────┐
 │               │
Policy v1      Policy v2
 │               │
Decision        Decision
 │               │
 └───────┬───────┘
         ↓
    Comparison
```

### Comparison Output

The comparison must identify:

| Dimension | What Changes |
|---|---|
| Outcome | APPROVE → REJECT, REVIEW → APPROVE, etc. |
| Reasons | New reasons added, existing reasons removed |
| Outputs | Credit limit changed, approved amount changed |
| Knockout behavior | Different knockouts triggered |
| Rule behavior | Different rules triggered |

### Comparison Constraints

- Uses the same assessment input for both policy versions
- Deterministic: same input + same policy versions = same comparison
- No historical datasets required
- No batch processing required
- No statistical analysis required

---

## Architecture

### Workspace as Client

The Workspace is a client of the existing Credora engine:

```text
Workspace (frontend)
    ↓
Credora API (existing)
    ↓
Assessment Service
    ↓
Policy Registry
    ↓
Policy Evaluator
    ↓
Decision + Evidence
```

### Key Constraint

The frontend must NOT implement decisioning logic. All evaluation happens in the backend. The Workspace renders results returned by the API.

### API Surface

The Workspace consumes existing API endpoints:

| Endpoint | Purpose |
|---|---|
| `POST /api/v1/assessments` | Create and evaluate an assessment |
| `GET /api/v1/assessments/{id}` | Retrieve assessment with decision |
| `GET /api/v1/assessments/{id}/decision` | Retrieve decision details |
| `GET /api/v1/assessments/{id}/evidence` | Retrieve evidence |

**Potential additional endpoints (future consideration):**

- `GET /api/v1/policies` — List available policies and versions
- `GET /api/v1/policies/{id}/versions` — List versions for a policy

These endpoints would serve policy metadata from the registry/database. They are not implemented in Phase 3 MVP but should be designed with this consumption pattern in mind.

### Relationship to /docs

```text
/docs
  → Developer API documentation
  → Interactive API client (Swagger UI)

/workspace
  → Decisioning Workspace
  → Developer and analyst-facing interface
```

They coexist. They serve different purposes.

---

## UI Technology

### Constraints

- The UI should be a separate presentation layer
- It must consume the Credora API
- It must not contain domain decisioning logic
- It must not duplicate policy evaluation
- It must support developer and risk/analyst workflows

### Decision

Do not make a final frontend framework decision in this document. The choice should be justified by:

- The existing repository structure
- Developer experience requirements
- The need for a lightweight, focused interface
- Compatibility with the Go backend

The Workspace is a separate application or sub-application, not embedded in the Go binary.

---

## Open Architectural Questions

These questions are unresolved and must be addressed before or during Phase 3 implementation.

### Policy Representation

1. **How should executable Go policies coexist with structured policy definitions?**

   The current model uses Go function closures. Structured definitions would need a parallel representation. Should they coexist, or should one become primary?

2. **Should structured policies be persisted independently from executable policies?**

   If structured definitions are introduced, should they live in the database alongside metadata, or remain in code?

3. **How can analyst-created rules remain deterministic?**

   If analysts eventually author rules, how does the system guarantee determinism without a DSL?

4. **How should conditions be represented without creating an unnecessarily complex DSL?**

   A minimal condition model (field/operator/value) might be sufficient. But what operators? How are compound conditions handled?

### Evidence and Traceability

5. **How should evidence references map to policy/rule execution?**

   The current model maps evidence to rule/knockout codes. Is this sufficient for structured policies?

### API Design

6. **Should the Workspace call existing assessment APIs or require new simulation-specific endpoints?**

   The existing `POST /api/v1/assessments` creates and persists an assessment. A simulation might want ephemeral evaluation without persistence. Should this be a separate endpoint?

7. **How should policy comparison be exposed by the API?**

   Should comparison be a single endpoint that accepts two policy versions? Or should the frontend call the assessment endpoint twice and compare client-side?

### Data Model

8. **Which data should be persisted for reproducibility?**

   Assessment inputs, decisions, evidence, and audit events are persisted. What about evaluation traces? Should the full trace be persisted for every assessment?

9. **What is the minimum financial input model required for DSR and limit assessment?**

   The current domain model has minimal financial inputs. DSR calculation and limit assessment may require additional input fields. What is the minimum viable set?

### Future

10. **How should future backtesting reuse the same evaluation engine?**

   The evaluation engine must accept both live and historical data. Should the engine be designed to accept pre-existing evidence (from providers) alongside live evaluation?

---

## Phase 3 MVP Acceptance Criteria

A Phase 3 MVP is complete when:

### Functional

1. A developer can open the Workspace.
2. An analyst can open the Workspace.
3. A user can see available policies and their versions.
4. A user can select a policy and version.
5. A user can enter assessment input (applicant, optional application, optional score).
6. A user can run the assessment.
7. Credora evaluates it using the backend engine.
8. A user sees APPROVE / REVIEW / REJECT.
9. A user sees structured reasons (code, description, value, threshold).
10. A user sees evidence entries.
11. A user sees credit limit / approved amount where applicable.
12. A user can compare the same assessment against two policy versions.
13. The comparison shows outcome, reasons, and output differences.

### Technical

14. The result is reproducible (same input + same policy = same output).
15. The frontend does not implement decisioning logic itself.
16. All evaluation happens in the backend.
17. The Workspace consumes the existing Credora API.
18. Policy metadata is visible to analysts without reading Go source.

### Non-Goals (Must NOT be in MVP)

19. No custom DSL introduced.
20. No policy editor.
21. No backtesting.
22. No authentication.
23. No RBAC.
24. No multi-tenancy.
25. No billing.
26. No provider management.
27. No BYOK.
28. No charts or analytics dashboards.
29. No CRM or LOS functionality.

---

## Future Direction

The likely evolution beyond Phase 3:

```text
Phase 3: Decisioning Workspace
  ↓
Phase 4: Structured Policy Definitions
  ↓
Phase 5: Policy Editor
  ↓
Phase 6: Policy Simulation (enhanced)
  ↓
Phase 7: Backtesting
  ↓
Phase 8: Portfolio Analytics
```

**Phase 4** would introduce a structured policy model (JSON/DB representation of rules, knockouts, score thresholds) that can be rendered in a UI and eventually edited by analysts.

**Phase 5** would build a policy editor on top of the structured model.

**Phase 7** would introduce backtesting against historical data using the same deterministic evaluation engine.

These are future phases. Phase 3 MVP does not require them.

---

## Source Code Reference

This specification is based on inspection of the following source files:

| File | Relevance |
|---|---|
| `services/engine/internal/domain/domain.go` | Core domain types (Assessment, Policy, Rule, Knockout, Decision, Evidence, EvaluationTrace) |
| `services/engine/internal/policy/evaluator.go` | Deterministic policy evaluation (Evaluate function, applyPrecedence, Validate) |
| `services/engine/internal/policy/registry.go` | In-memory policy registry (Register, Get, Has) |
| `services/engine/internal/policy/defaults.go` | Default policy definitions (personal-loan v1 with knockouts, rules, score thresholds) |
| `services/engine/internal/assessment/service.go` | Assessment lifecycle service (Create, GetByID, execute) |
| `services/engine/internal/repository/repository.go` | Repository interfaces (AssessmentRepository, PolicyRepository) |
| `services/engine/internal/repository/sqlite.go` | SQLite persistence implementation |
| `services/engine/internal/http/handler.go` | HTTP API handlers (CreateAssessment, GetAssessment, GetDecision, GetEvidence) |
| `services/engine/internal/http/dto.go` | HTTP DTOs (CreateAssessmentRequest, AssessmentResponse, DecisionResponse, etc.) |
| `services/engine/cmd/server/main.go` | Application wiring (database, registry, service, handler, routes) |
| `docs/domain.md` | Domain model documentation |
| `docs/openapi.yaml` | OpenAPI specification |
