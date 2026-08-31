// Package domain defines the core types for Credora's credit decisioning engine.
//
// All domain types are independent of HTTP, database, Temporal, and external providers.
// The evaluation model is deterministic: the same assessment input and policy version
// must always produce the same decision.
package domain

import "time"

// DecisionOutcome represents the business decision produced by policy evaluation.
type DecisionOutcome int

const (
	DecisionApprove DecisionOutcome = iota
	DecisionReview
	DecisionReject
)

func (d DecisionOutcome) String() string {
	switch d {
	case DecisionApprove:
		return "APPROVE"
	case DecisionReview:
		return "REVIEW"
	case DecisionReject:
		return "REJECT"
	default:
		return "UNKNOWN"
	}
}

// RuleOutcome defines what happens when a rule's condition matches.
type RuleOutcome int

const (
	RuleOutcomePass RuleOutcome = iota
	RuleOutcomeReview
	RuleOutcomeReject
)

// ConditionResult captures the structured output of a condition evaluation.
// It carries the actual value evaluated, the threshold or reference value,
// whether the condition matched, and a human-readable description.
// This supports explainability and backtesting without requiring a DSL.
type ConditionResult struct {
	Actual    interface{} // actual value evaluated
	Threshold interface{} // reference/threshold value (nil if not applicable)
	Matched   bool        // whether the condition triggered
	Detail    string      // human-readable description of the check performed
}

// DecisionReason explains why a specific decision was produced.
// Each reason references a stable code, the actual value observed,
// the threshold it was compared against, and an evidence reference.
type DecisionReason struct {
	Code        string
	Description string
	Value       interface{}
	Threshold   interface{}
	EvidenceRef string
}

// Evidence captures the provenance of a value used in decisioning.
// Evidence is first-class: every decision reason should be traceable
// to supporting evidence.
type Evidence struct {
	Source      string
	Field       string
	Value       interface{}
	RetrievedAt time.Time
	Reference   string
}

// DecisionOutputs contains policy-produced values such as credit limit
// or approved amount. These are outputs of the decision, not inputs.
// A limit-assessment policy may produce a CreditLimit without a RequestedAmount.
// A loan-application policy may produce an ApprovedAmount based on the request.
type DecisionOutputs struct {
	CreditLimit    *int64 `json:"credit_limit,omitempty"`
	ApprovedAmount *int64 `json:"approved_amount,omitempty"`
}

// Decision is the final output of policy evaluation.
// It contains the outcome, structured reasons, and policy metadata.
// The full evaluation trace is returned separately for analysis.
type Decision struct {
	Outcome       DecisionOutcome
	Reasons       []DecisionReason
	Outputs       *DecisionOutputs
	PolicyID      string
	PolicyVersion int
}

// KnockoutEvaluation records the result of evaluating one knockout condition.
type KnockoutEvaluation struct {
	Code        string
	Description string
	Result      ConditionResult
	ReasonCode  string
	ReasonDesc  string
}

// RuleEvaluation records the result of evaluating one rule condition.
type RuleEvaluation struct {
	Code        string
	Description string
	Result      ConditionResult
	Outcome     RuleOutcome
	ReasonCode  string
	ReasonDesc  string
}

// EvaluationTrace captures the full evaluation path for analysis and backtesting.
// It preserves all knockouts evaluated (including non-triggered), all rules
// evaluated, and all evidence. This makes future backtesting possible even
// when live evaluation short-circuits on the first knockout.
type EvaluationTrace struct {
	Knockouts []KnockoutEvaluation
	Rules     []RuleEvaluation
	Evidence  []Evidence
}

// Applicant represents the individual applying for credit.
type Applicant struct {
	ID   string
	Name string
	Age  int
}

// Application represents a specific credit application.
// Application is optional: an assessment may be performed without one
// (e.g., limit assessment, pre-screening, periodic reassessment).
// RequestedAmount is optional: not all assessment types require a request.
// All monetary values use the smallest currency unit (e.g., cents, sen).
type Application struct {
	ID              string
	RequestedAmount *int64
	Purpose         string
}

// CreditScore represents a quantitative credit assessment.
// Credora does not implement proprietary scoring.
// The score is an input: it may come from a provider,
// a customer-defined calculation, or a deterministic rule.
//
// The score scale, ranges, and threshold meanings are policy-specific.
// Credora does not assume a universal credit score scale.
type CreditScore struct {
	Value    int
	Provider string
}

// AssessmentStatus represents the lifecycle state of an assessment.
type AssessmentStatus int

const (
	AssessmentPending AssessmentStatus = iota
	AssessmentRunning
	AssessmentCompleted
	AssessmentFailed
)

func (s AssessmentStatus) String() string {
	switch s {
	case AssessmentPending:
		return "PENDING"
	case AssessmentRunning:
		return "RUNNING"
	case AssessmentCompleted:
		return "COMPLETED"
	case AssessmentFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// Assessment represents one execution of a credit decisioning process.
// It carries all inputs needed for policy evaluation, plus lifecycle state.
//
// Application is optional: an assessment may evaluate an applicant directly
// (e.g., limit assessment, pre-screening) without an application context.
// MonthlyIncome and MonthlyObligations are optional: not all assessment types
// require an application context with financial facts (e.g., limit assessment
// may have financial facts directly on the assessment).
type Assessment struct {
	ID          string
	Applicant   Applicant
	Application *Application // optional
	Score       *CreditScore

	MonthlyIncome      *int64 // optional, smallest currency unit
	MonthlyObligations *int64 // optional, smallest currency unit

	Status        AssessmentStatus
	Error         string
	Decision      *Decision
	Evidence      []Evidence
	PolicyID      string
	PolicyVersion int

	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// Rule defines a deterministic policy condition.
// A rule triggers when its Condition returns a ConditionResult with Matched=true.
// The Outcome determines what happens when the rule triggers.
type Rule struct {
	Code        string
	Description string
	Condition   func(Assessment) ConditionResult
	Outcome     RuleOutcome
	ReasonCode  string
	ReasonDesc  string
}

// Knockout defines a condition that makes an application ineligible.
// A knockout is distinct from an ordinary rule: it always produces REJECT.
type Knockout struct {
	Code        string
	Description string
	Condition   func(Assessment) ConditionResult
	ReasonCode  string
	ReasonDesc  string
}

// ScoreThresholds defines score-based decision boundaries.
// When present and a score is available, these thresholds are used
// after rule evaluation to determine the final decision.
//
// The score scale and threshold meanings are policy-specific.
// Credora does not assume a universal credit score scale.
// Different policies may use different scales (e.g., 300-850, 0-100, 1-10).
type ScoreThresholds struct {
	ApproveThreshold int
	ReviewThreshold  int
}

// Policy represents a versioned collection of decisioning logic.
// Policy evaluation must be reproducible: the same policy version
// applied to the same assessment must always produce the same result.
type Policy struct {
	ID              string
	Version         int
	Name            string
	Description     string
	Knockouts       []Knockout
	Rules           []Rule
	ScoreThresholds *ScoreThresholds
	// Outputs computes policy-produced values (e.g., credit limit,
	// approved amount) for the decision. It is optional: policies that
	// do not produce outputs leave it nil.
	Outputs func(Assessment) *DecisionOutputs
	// DefaultOutcome is used when no knockouts, rules, or score thresholds
	// determine the outcome. This prevents silent APPROVE when the policy
	// cannot establish an approval condition. Policies should set this
	// explicitly (e.g., DecisionReview for cautious default).
	DefaultOutcome DecisionOutcome
}
