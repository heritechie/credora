package policy

import (
	"testing"

	"credora/internal/domain"
)

func int64Ptr(v int64) *int64 { return &v }

func defaultAssessment() domain.Assessment {
	return domain.Assessment{
		ID:        "a-001",
		Applicant: domain.Applicant{ID: "app-001", Name: "Test Applicant", Age: 30},
	}
}

// TestDefaultPolicy_NoFinancialsDoesNotPanic verifies that the default
// personal-loan policy evaluates an assessment without financial facts
// without panicking. MonthlyIncome and MonthlyObligations are optional.
func TestDefaultPolicy_NoFinancialsDoesNotPanic(t *testing.T) {
	pol := defaultPersonalLoanV1()
	assessment := defaultAssessment()

	if _, _, err := Evaluate(assessment, pol); err != nil {
		t.Fatalf("Evaluate without financials: %v", err)
	}
}

// TestDefaultPolicy_HighDSRTriggersReject verifies the HIGH_DSR knockout.
// DSR = monthly_obligations / monthly_income. Above 70% the assessment is rejected.
func TestDefaultPolicy_HighDSRTriggersReject(t *testing.T) {
	pol := defaultPersonalLoanV1()
	assessment := defaultAssessment()
	assessment.MonthlyIncome = int64Ptr(10000000)
	assessment.MonthlyObligations = int64Ptr(8000000) // 80% DSR
	assessment.Score = &domain.CreditScore{Value: 720, Provider: "mock-credit-bureau"}

	decision, _, err := Evaluate(assessment, pol)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if decision.Outcome != domain.DecisionReject {
		t.Fatalf("expected REJECT, got %s", decision.Outcome.String())
	}
	if len(decision.Reasons) != 1 || decision.Reasons[0].Code != "HIGH_DSR" {
		t.Fatalf("expected HIGH_DSR reason, got %+v", decision.Reasons)
	}
}

// TestDefaultPolicy_LimitAssessmentOutputs verifies that a limit assessment
// (no application) produces a credit limit and no approved amount.
func TestDefaultPolicy_LimitAssessmentOutputs(t *testing.T) {
	pol := defaultPersonalLoanV1()
	assessment := defaultAssessment()
	assessment.MonthlyIncome = int64Ptr(10000000)
	assessment.MonthlyObligations = int64Ptr(3000000)
	assessment.Score = &domain.CreditScore{Value: 720, Provider: "mock-credit-bureau"}

	decision, _, err := Evaluate(assessment, pol)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if decision.Outputs == nil {
		t.Fatal("expected outputs, got nil")
	}
	if decision.Outputs.CreditLimit == nil || *decision.Outputs.CreditLimit != 20000000 {
		t.Errorf("expected credit limit 20000000, got %v", decision.Outputs.CreditLimit)
	}
	if decision.Outputs.ApprovedAmount != nil {
		t.Errorf("expected no approved amount for limit assessment, got %v", *decision.Outputs.ApprovedAmount)
	}
}

// TestDefaultPolicy_LoanApplicationOutputs verifies that a loan application
// produces an approved amount equal to min(requested_amount, credit_limit).
func TestDefaultPolicy_LoanApplicationOutputs(t *testing.T) {
	pol := defaultPersonalLoanV1()
	assessment := defaultAssessment()
	assessment.MonthlyIncome = int64Ptr(10000000) // credit limit 20000000
	assessment.MonthlyObligations = int64Ptr(3000000)
	assessment.Score = &domain.CreditScore{Value: 720, Provider: "mock-credit-bureau"}
	assessment.Application = &domain.Application{
		ID:              "app-001",
		RequestedAmount: int64Ptr(15000000),
		Purpose:         "working_capital",
	}

	decision, _, err := Evaluate(assessment, pol)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if decision.Outputs == nil {
		t.Fatal("expected outputs, got nil")
	}
	if decision.Outputs.CreditLimit == nil || *decision.Outputs.CreditLimit != 20000000 {
		t.Errorf("expected credit limit 20000000, got %v", decision.Outputs.CreditLimit)
	}
	if decision.Outputs.ApprovedAmount == nil || *decision.Outputs.ApprovedAmount != 15000000 {
		t.Errorf("expected approved amount 15000000, got %v", decision.Outputs.ApprovedAmount)
	}
}

// TestDefaultPolicy_LoanApplicationCappedApproved verifies the approved amount
// is capped at the credit limit when the request exceeds it.
func TestDefaultPolicy_LoanApplicationCappedApproved(t *testing.T) {
	pol := defaultPersonalLoanV1()
	assessment := defaultAssessment()
	assessment.MonthlyIncome = int64Ptr(10000000) // credit limit 20000000
	assessment.MonthlyObligations = int64Ptr(3000000)
	assessment.Score = &domain.CreditScore{Value: 720, Provider: "mock-credit-bureau"}
	assessment.Application = &domain.Application{
		ID:              "app-001",
		RequestedAmount: int64Ptr(25000000),
		Purpose:         "working_capital",
	}

	decision, _, err := Evaluate(assessment, pol)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if decision.Outputs == nil {
		t.Fatal("expected outputs, got nil")
	}
	if decision.Outputs.ApprovedAmount == nil || *decision.Outputs.ApprovedAmount != 20000000 {
		t.Errorf("expected approved amount capped at 20000000, got %v", decision.Outputs.ApprovedAmount)
	}
}

// TestDefaultPolicy_NoIncomeNoOutputs verifies that a policy with no income
// does not produce outputs.
func TestDefaultPolicy_NoIncomeNoOutputs(t *testing.T) {
	pol := defaultPersonalLoanV1()
	assessment := defaultAssessment()
	assessment.Score = &domain.CreditScore{Value: 720, Provider: "mock-credit-bureau"}

	decision, _, err := Evaluate(assessment, pol)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if decision.Outputs != nil {
		t.Errorf("expected no outputs without income, got %+v", decision.Outputs)
	}
}
