# Provider Scenario Testing

## Overview

Provider Scenario Testing is a Credora capability for generating, inspecting, and executing realistic synthetic provider scenarios to improve development and testing of credit decisioning workflows. This addresses the gap where provider sandboxes allow basic happy-path testing but lack wide-ranging synthetic scenario coverage.

Credora is **not** a credit bureau, proprietary scorer, or API mocking platform. Provider scenario testing is a development and testing aid that generates synthetic data based on provider contract fidelity — it does not replace production APIs or claim identical production behavior.

---

## Problem

Provider sandboxes typically offer:
- Basic happy-path testing only
- Limited scenario coverage
- No systematic way to generate synthetic test cases
- Difficult to simulate edge cases, failures, and varied data states

This capability enables developers and credit analysts to test decisioning workflows across a broad range of provider behaviors without depending on production API availability or specific test data.

---

## Core Concepts

### Provider Template

A **Provider Template** defines the contract and data structure for a provider category. Templates are based on actual provider documentation (e.g., FDC, PEFINDO examples) but are **not dependencies** — they serve as semantic references for scenario generation.

Provider templates define:
- **Category**: Credit Provider, Identity Provider, Fraud Provider, Income Provider
- **Endpoint**: HTTP method and path pattern
- **Request schema**: Fields expected in the request
- **Response schema**: Fields expected in the response, including types and constraints
- **HTTP status codes**: Expected success/error codes
- **Error format**: Structure of error responses
- **Authentication**: How requests are authenticated (conceptual, not stored)

> **Important**: Provider-specific details remain isolated inside adapters/templates. The policy layer must not depend on FDCResponse, PEFINDOResponse, or provider-specific types.

### Synthetic Scenario

A **Synthetic Scenario** is a generated test case that represents realistic provider behaviors. Scenarios are defined as **semantic facts** rather than arbitrary JSON, ensuring they map to meaningful credit decisioning conditions.

Scenario categories:

| Category | Description |
|---|---|
| **Business/data scenarios** | Clean credit, thin file, delinquency, high DSR, etc. |
| **Provider behavior scenarios** | HTTP 200, HTTP 4xx, HTTP 5xx, timeout, malformed response, rate limit |

Each scenario maps to normalized provider data that flows through the Credora pipeline (provider adapter → score calculation → policy evaluation → decision).

### Provider-Faithful Response

A provider-faithful synthetic response satisfies **contract fidelity** and **data fidelity**:

**Contract fidelity:**
- Endpoint matches the provider template
- HTTP status code is valid (2xx, 4xx, 5xx)
- Response schema conforms to the provider template
- Error format matches the provider template

**Data fidelity:**
- Synthetic values are realistic for the credit domain
- Values are not arbitrary — they correspond to meaningful credit scenarios
- The same scenario generative seed produces consistent results

### Scenario Generator

The **Scenario Generator** is the mechanism for producing synthetic scenarios. Key design constraints:

- LLM may assist scenario generation but **MUST NOT** make credit decisions, execute policy logic, or replace the policy evaluator
- Scenarios are validated against provider templates before acceptance
- Generated scenarios are explicitly distinguished from real provider data
- Scenario definitions are versioned and reproducible

> **Constraint**: The scenario generator does not implement or encapsulate credit policy logic. It produces data that flows through the existing policy evaluation pipeline.

### Provider Adapter

The **Provider Adapter** is the existing Credora infrastructure that:
- Constructs provider-specific requests
- Maps provider responses to normalized domain types
- Handles provider-specific error handling
- Normalizes provider data into Credora domain types (e.g., monthly_obligation, credit_score, etc.)

The rest of Credora should not depend on provider-specific response formats. The adapter is the only component that understands provider-specific details.

---

## Workflow

The Provider Scenario Testing workflow consists of the following steps:

1. **Provider Selection**: Developer selects a provider category (credit, identity, fraud, income)
2. **Template Reference**: System references the provider template for the selected category
3. **Scenario Generation**: Synthetic scenario is generated (LLM-assisted with constraints)
4. **Scenario Validation**: Scenario is validated against the provider template (contract + data fidelity)
5. **Mock Response Creation**: Validated scenario produces a provider-faithful mock response
6. **Provider Adapter Execution**: Mock response flows through the provider adapter
7. **Normalized Provider Data**: Adapter outputs normalized provider data fields
8. **Credit Score Calculation**: Provider-derived score (or customer scoring model) is calculated
9. **Policy Evaluation**: Policy is evaluated with the provider data as input
10. **Decision Production**: Structured decision (APPROVE/REVIEW/REJECT) with reasons and evidence
11. **Artifact Saving**: Decision, evidence, and audit artifacts are persisted

> **Note**: Steps 1-5 may be assisted or automated; steps 6-11 use the existing Credora engine.

---

## Testing Levels

Provider Scenario Testing supports three levels of testing:

| Level | Description |
|---|---|
| **Provider Scenario Testing** | Generate and execute synthetic provider scenarios; verify provider adapter behavior across varied inputs |
| **Assessment Simulation** | Run full assessments with synthetic provider data; verify end-to-end decisioning pipeline |
| **Policy Comparison** | Compare how the same assessment behaves under different policy versions with synthetic provider data |

---

## Non-Goals

The following are explicitly NOT part of this capability:

- Not a replacement for production provider APIs
- Not a credit bureau or credit scoring service
- Not generic API mocking (scenarios are credit-domain-specific)
- Not provider-specific dependency — templates are semantic references only
- Not ML model training or proprietary scoring
- Not authentication or credential management
- Not real-time provider data fetching
- Not production API proxy or routing

---

## LLM Constraints

An LLM may assist in scenario generation but must adhere to these constraints:

- **MUST NOT** make credit decisions (APPROVE/REVIEW/REJECT)
- **MUST NOT** execute policy logic or rule evaluation
- **MUST NOT** replace the policy evaluator
- **MUST NOT** generate arbitrary JSON without schema constraints
- **MUST** generate scenarios that map to semantic credit facts
- **MUST** respect provider template constraints (endpoint, schema, status codes)
- **MUST** clearly distinguish synthetic data from real provider data
- **MUST** not commit secrets, API keys, or credentials

---

## Example Scenarios

### Business/Data Scenarios

| Scenario | Description | Expected Normalized Data |
|---|---|---|
| **Clean credit** | Ideal credit profile | credit_score: 750, monthly_obligation: 5000, dsr: 0.25 |
| **Thin file** | Limited credit history | credit_score: 620, monthly_obligation: 3000, dsr: 0.45 |
| **Delinquency** | Recent payment delinquency | credit_score: 580, monthly_obligation: 4000, dsr: 0.50, late_payments: 3 |
| **High DSR** | Debt service ratio exceeds threshold | credit_score: 680, monthly_obligation: 8000, income: 8000, dsr: 1.0 |
| **Unemployed** | No verified income | credit_score: 600, monthly_obligation: 2000, income: 0, dsr: null |

### Provider Behavior Scenarios

| Scenario | HTTP Status | Description |
|---|---|---|
| **Successful response** | 200 | Normal provider response |
| **Not found** | 404 | Provider resource not found |
| **Authentication error** | 401 | Invalid provider credentials |
| **Validation error** | 422 | Provider-request validation failed |
| **Server error** | 500 | Provider internal error |
| **Rate limited** | 429 | Too many requests |
| **Timeout** | N/A | Request timed out |

---

## Developer Workflow

1. Select provider category from available templates
2. Generate synthetic scenario (LLM-assisted or manual)
3. Validate scenario against provider template
4. Execute scenario through provider adapter
5. Observe normalized provider data flow
6. Run full assessment with synthetic data
7. Inspect decision, reasons, and evidence
8. Save artifacts for later review

---

## Differentiation from Traditional Sandboxes

| Aspect | Traditional Sandbox | Provider Scenario Testing |
|---|---|---|
| **Scope** | Specific provider's production-like environment | Synthetic scenarios across provider categories |
| **Data** | Real provider responses | Synthetic, realistic data |
| **Scenarios** | Limited to what provider offers | Wide-ranging: business scenarios + provider behaviors |
| **Generation** | Manual test data creation | Dynamic, LLM-assisted with constraints |
| **Policy testing** | Happy-path only | Full pipeline: provider → score → policy → decision |
| **Fidelity** | Exact provider API | Contract + data fidelity (not identical to production) |

---

## Open Design Questions

The following design questions are noted for future resolution:

1. **Template representation**: How are provider templates structured? (JSON schema, Go structs, hybrid?)
2. **Documentation sourcing**: Where do provider docs come from? (Customer-provided, curated, example-based?)
3. **Contract versioning**: How are provider contract versions tracked and versioned?
4. **Scenario validation**: What validates a scenario against a template? (JSON schema, Go type checks, hybrid?)
5. **Synthetic data consistency**: How is consistency ensured across generated scenarios?
6. **Renderer design**: What format do rendered scenarios use? (Markdown, structured data, interactive?)
7. **LLM constraints**: Additional constraints beyond "must not make credit decisions"?
8. **Sensitive documentation handling**: How are provider docs with sensitive information handled?

These questions are documented explicitly but do not block the initial specification.

---

## Document Metadata

- **Location**: `docs/product/provider-scenario-testing.md`
- **Status**: Product specification — no code, engine, or Workspace modifications
- **Related**: `docs/phases/phase-3.md`, `AGENTS.md`, `docs/domain.md`
- **Phase**: Defined as product direction for future Credora development

---

## Change History

| Date | Change |
|---|---|
| 2026-08-30 | Initial product specification created |

---