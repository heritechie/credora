# Domain Model

This document defines Credora's core domain concepts and evaluation model.

These are Credora's documented domain definitions. They do not represent universal industry standards. Where methodology is policy-specific, this is explicitly noted.

## Core Concepts

### Assessment

An Assessment represents one execution of a credit decisioning process.

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Unique assessment identifier |
| Applicant | Applicant | The individual applying for credit |
| Application | Application | The specific credit application |
| Score | *CreditScore | Optional credit score input |

### Applicant

An Applicant represents the individual applying for credit.

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Unique applicant identifier |
| Name | string | Applicant name |
| Age | int | Applicant age |

### Application

An Application represents a specific credit request.

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Unique application identifier |
| RequestedAmount | float64 | Amount requested |
| Purpose | string | Purpose of the credit |

### CreditScore

A CreditScore is a quantitative credit assessment. Credora does not implement proprietary scoring. The score is an input that may come from:
- an external provider
- a customer-defined calculation
- a deterministic rule

| Field | Type | Description |
|-------|------|-------------|
| Value | int | Numeric score value |
| Provider | string | Source of the score |

**Important:** The score scale, ranges, and threshold meanings are policy-specific. Credora does not assume a universal credit score scale. Different policies may use different scales (e.g., 300-850, 0-100, 1-10). The `ScoreThresholds` values are interpreted relative to the scale used by the policy.

### ConditionResult

A ConditionResult captures the structured output of a condition evaluation. It carries the actual value evaluated, the threshold or reference value, whether the condition matched, and a human-readable description. This supports explainability and backtesting without requiring a DSL.

| Field | Type | Description |
|-------|------|-------------|
| Actual | interface{} | Actual value evaluated |
| Threshold | interface{} | Reference/threshold value (nil if not applicable) |
| Matched | bool | Whether the condition triggered |
| Detail | string | Human-readable description of the check |

### Policy

A Policy represents a versioned collection of decisioning logic. Policy evaluation must be reproducible: the same policy version applied to the same assessment must always produce the same result.

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Policy identifier |
| Version | int | Policy version number |
| Name | string | Human-readable policy name |
| Description | string | Human-readable policy description |
| Knockouts | []Knockout | Knockout conditions |
| Rules | []Rule | Decisioning rules |
| ScoreThresholds | *ScoreThresholds | Optional score thresholds |
| DefaultOutcome | DecisionOutcome | Outcome when no conditions determine the result |

**DefaultOutcome** is used when no knockouts, rules, or score thresholds determine the outcome. This prevents silent APPROVE when the policy cannot establish an approval condition. Policies should set this explicitly (e.g., `DecisionReview` for cautious default).

#### Policy Immutability

Once a policy version has been used by an assessment, its decisioning semantics must never change. Create new versions instead. This ensures:
- Historical assessments remain reproducible
- Policy evaluation is deterministic
- Backtesting is valid

#### Policy Registry

Executable policy definitions are maintained in Go code via a `Policy` registry. The registry holds the actual decisioning logic (knockouts, rules, score thresholds). Policy metadata (name, description) may be persisted separately in the database.

The assessment service resolves policies from the registry by ID and version, rather than accepting a hardcoded policy parameter.

#### PolicyRepository

The `PolicyRepository` persists policy metadata in the database for audit purposes. It stores:
- Policy ID and version
- Name and description
- Creation timestamp

The executable policy logic is NOT stored in the database—it lives in the Go registry. This design:
- Avoids serializing Go function closures to the database
- Keeps policy definitions deterministic and version-controlled
- Supports the same databases as the assessment repository (SQLite, PostgreSQL)

### Rule

A Rule defines a deterministic policy condition. A rule triggers when its Condition returns a ConditionResult with Matched=true. The Outcome determines what happens when the rule triggers.

| Field | Type | Description |
|-------|------|-------------|
| Code | string | Stable identifier |
| Description | string | Human-readable description |
| Condition | func(Assessment) ConditionResult | Evaluation function |
| Outcome | RuleOutcome | What happens when triggered |
| ReasonCode | string | Stable reason code |
| ReasonDesc | string | Reason description |

Rule outcomes:
- `RuleOutcomePass` - assessment passes this rule
- `RuleOutcomeReview` - assessment needs review
- `RuleOutcomeReject` - assessment is rejected

### Knockout

A Knockout defines a condition that makes an application ineligible. A knockout is distinct from an ordinary rule: it always produces REJECT.

| Field | Type | Description |
|-------|------|-------------|
| Code | string | Stable identifier |
| Description | string | Human-readable description |
| Condition | func(Assessment) ConditionResult | Evaluation function |
| ReasonCode | string | Stable reason code |
| ReasonDesc | string | Reason description |

Example knockout codes (generic, not proprietary):
- `AGE_BELOW_MINIMUM`
- `DSR_ABOVE_MAXIMUM`
- `IDENTITY_VERIFICATION_FAILED`

### ScoreThresholds

ScoreThresholds defines score-based decision boundaries. Used when no knockouts or rules trigger and a score is available.

| Field | Type | Description |
|-------|------|-------------|
| ApproveThreshold | int | Score >= this → APPROVE |
| ReviewThreshold | int | Score >= this → REVIEW |

Scores below ReviewThreshold → REJECT.

**Important:** The score scale and threshold meanings are policy-specific. Credora does not assume a universal credit score scale. Different policies may use different scales. The threshold values are interpreted relative to the scale used by the policy.

### Decision

A Decision is the final output of policy evaluation.

| Field | Type | Description |
|-------|------|-------------|
| Outcome | DecisionOutcome | APPROVE, REVIEW, or REJECT |
| Reasons | []DecisionReason | Structured reasons |
| PolicyID | string | Policy that produced this decision |
| PolicyVersion | int | Policy version used |

The full evaluation trace (all knockouts evaluated, all rules evaluated, all evidence) is returned separately as an `EvaluationTrace`.

### DecisionReason

A DecisionReason explains why a specific decision was produced.

| Field | Type | Description |
|-------|------|-------------|
| Code | string | Stable reason code |
| Description | string | Human-readable description |
| Value | interface{} | Actual value observed (from ConditionResult) |
| Threshold | interface{} | Threshold compared against (from ConditionResult) |
| EvidenceRef | string | Reference to supporting evidence |

### Evidence

Evidence captures the provenance of a value used in decisioning. Evidence is first-class: every decision reason should be traceable to supporting evidence.

| Field | Type | Description |
|-------|------|-------------|
| Source | string | What produced this evidence ("rule" or "knockout") |
| Field | string | What field this relates to (rule/knockout code) |
| Value | interface{} | The evaluation result (Matched bool) |
| RetrievedAt | time.Time | When this evidence was produced |
| Reference | string | ID/code of the source |

### EvaluationTrace

An EvaluationTrace captures the full evaluation path for analysis and backtesting. It preserves all knockouts evaluated (including non-triggered), all rules evaluated, and all evidence.

| Field | Type | Description |
|-------|------|-------------|
| Knockouts | []KnockoutEvaluation | All knockout evaluations |
| Rules | []RuleEvaluation | All rule evaluations |
| Evidence | []Evidence | All evidence collected |

### KnockoutEvaluation

Records the result of evaluating one knockout condition.

| Field | Type | Description |
|-------|------|-------------|
| Code | string | Knockout code |
| Description | string | Knockout description |
| Result | ConditionResult | The condition evaluation result |
| ReasonCode | string | Reason code if triggered |
| ReasonDesc | string | Reason description if triggered |

### RuleEvaluation

Records the result of evaluating one rule condition.

| Field | Type | Description |
|-------|------|-------------|
| Code | string | Rule code |
| Description | string | Rule description |
| Result | ConditionResult | The condition evaluation result |
| Outcome | RuleOutcome | What happens if matched |
| ReasonCode | string | Reason code if matched |
| ReasonDesc | string | Reason description if matched |

## DSR (Debt Service Ratio)

DSR represents the relationship between debt-service obligations and the income measure used by a credit policy.

Credora does not hard-code a universal DSR formula. Different policies may define:
- income definition
- debt obligation definition
- period (monthly, annual)
- treatment of edge cases

The implementation keeps these assumptions explicit. DSR calculation is policy-specific and not implemented in the core engine.

## Evaluation Order

The policy evaluator follows a deterministic evaluation order:

1. **Validate assessment input** - ensure required fields are present
2. **Evaluate all knockout conditions** - all are evaluated, any trigger produces REJECT
3. **Evaluate all ordinary policy rules** - all are evaluated, collect triggered outcomes
4. **Apply score thresholds** - if no knockouts or reject rules triggered
5. **Apply precedence** - determine final decision from all evaluations
6. **Collect evidence** - all evaluations produce evidence
7. **Produce structured decision** - outcome with reasons
8. **Produce evaluation trace** - complete record of all evaluations

All conditions are evaluated even when the outcome is already determined. This ensures the EvaluationTrace is complete for analysis and backtesting.

## Precedence

When multiple conditions trigger, the following precedence applies:

1. **Knockouts** - highest priority, always produce REJECT
2. **Rules with REJECT outcome** - produce REJECT
3. **Rules with REVIEW outcome** - produce REVIEW
4. **Score thresholds** - used when no knockouts or rules trigger
5. **DefaultOutcome** - used when no conditions determine the result

Evidence is collected for all evaluations, even when they do not trigger. This ensures full traceability.

## Determinism

The evaluator is deterministic: the same assessment input and policy version must always produce the same decision. This property is:
- required for reproducibility
- necessary for backtesting
- essential for explainability

## Constraints

For this phase:
- Rule and knockout conditions are Go functions returning ConditionResult (no DSL)
- Policy metadata is persisted in the database (SQLite/PostgreSQL)
- Executable policy logic is in Go code (policy registry)
- No provider integrations
- No authentication
- No multi-tenancy
- No custom expression languages
