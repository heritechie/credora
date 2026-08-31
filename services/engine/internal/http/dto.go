// Package http implements the HTTP API layer for the credit decisioning engine.
//
// Handlers are thin: they validate requests, call the application service,
// and map domain results to HTTP responses. They must not contain
// provider logic, policy logic, scoring logic, or database queries.
package http

import "time"

// CreateAssessmentRequest is the API contract for creating an assessment.
// Applicant is required. Application is optional. Policy is optional with defaults.
// MonthlyIncome and MonthlyObligations are optional financial facts used by
// policies that evaluate DSR or compute credit limits.
// All monetary values use the smallest currency unit (e.g., cents, sen).
type CreateAssessmentRequest struct {
	Applicant   ApplicantDTO    `json:"applicant"`
	Application *ApplicationDTO `json:"application,omitempty"`
	Score       *ScoreDTO       `json:"score,omitempty"`
	Policy      *PolicyRequest  `json:"policy,omitempty"`

	MonthlyIncome      *int64 `json:"monthly_income,omitempty"`
	MonthlyObligations *int64 `json:"monthly_obligations,omitempty"`
}

// PolicyRequest represents the policy identifiers in API requests.
// If omitted, defaults to "personal-loan" v1.
type PolicyRequest struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

// ApplicantDTO represents the applicant in API requests/responses.
type ApplicantDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// ApplicationDTO represents the application in API requests/responses.
// RequestedAmount is optional: not all assessment types require a request.
// All monetary values use the smallest currency unit (e.g., cents, sen).
type ApplicationDTO struct {
	ID              string `json:"id"`
	RequestedAmount *int64 `json:"requested_amount,omitempty"`
	Purpose         string `json:"purpose"`
}

// ScoreDTO represents a credit score in API requests/responses.
type ScoreDTO struct {
	Value    int    `json:"value"`
	Provider string `json:"provider"`
}

// AssessmentResponse is the API response for an assessment.
type AssessmentResponse struct {
	ID          string            `json:"id"`
	Status      string            `json:"status"`
	Applicant   ApplicantDTO      `json:"applicant"`
	Application *ApplicationDTO   `json:"application,omitempty"`
	Score       *ScoreDTO         `json:"score,omitempty"`
	Policy      PolicyDTO         `json:"policy"`
	Decision    *DecisionResponse `json:"decision,omitempty"`
	Error       string            `json:"error,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
}

// AssessmentListItem is a minimal summary of an assessment for listing.
type AssessmentListItem struct {
	ID          string        `json:"id"`
	Status      string        `json:"status"`
	Policy      PolicyDTO     `json:"policy"`
	Decision    *DecisionList `json:"decision,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
}

// DecisionList is a minimal decision summary for listing.
type DecisionList struct {
	Outcome string `json:"outcome"`
}

// AssessmentListResponse is the API response for listing assessments.
type AssessmentListResponse struct {
	Items []AssessmentListItem `json:"items"`
}

// PolicyDTO represents policy metadata in API responses.
type PolicyDTO struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

// PolicyListItem is a minimal policy summary for listing.
type PolicyListItem struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
}

// PolicyListResponse is the API response for listing policies.
type PolicyListResponse struct {
	Items []PolicyListItem `json:"items"`
}

// DecisionResponse is the API response for a decision.
type DecisionResponse struct {
	Outcome string              `json:"outcome"`
	Reasons []ReasonResponse    `json:"reasons"`
	Outputs *DecisionOutputsDTO `json:"outputs,omitempty"`
	Policy  PolicyDTO           `json:"policy"`
}

// DecisionOutputsDTO contains policy-produced values such as credit limit
// or approved amount. These are outputs of the decision, not inputs.
type DecisionOutputsDTO struct {
	CreditLimit    *int64 `json:"credit_limit,omitempty"`
	ApprovedAmount *int64 `json:"approved_amount,omitempty"`
}

// ReasonResponse is the API response for a decision reason.
type ReasonResponse struct {
	Code        string      `json:"code"`
	Description string      `json:"description"`
	Value       interface{} `json:"value,omitempty"`
	Threshold   interface{} `json:"threshold,omitempty"`
	EvidenceRef string      `json:"evidence_ref,omitempty"`
}

// EvidenceResponse is the API response for an evidence entry.
type EvidenceResponse struct {
	Source      string      `json:"source"`
	Field       string      `json:"field"`
	Value       interface{} `json:"value"`
	RetrievedAt time.Time   `json:"retrieved_at"`
	Reference   string      `json:"reference"`
}

// ErrorResponse is the API response for errors.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}
