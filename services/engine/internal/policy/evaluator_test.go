package policy

import (
	"testing"

	"credora/internal/domain"
)

func ptr[T any](v T) *T { return &v }

func TestEvaluate_Approve(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Alice",
			Age:  30,
		},
		Application: &domain.Application{
			ID:              "app1",
			RequestedAmount: ptr(int64(50000)),
			Purpose:         "personal",
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "AGE_MINIMUM",
				Description: "Applicant must be at least 18",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 18,
						Matched:   a.Applicant.Age < 18,
						Detail:    "age < 18",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Applicant is under 18",
			},
		},
	}

	decision, trace, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionApprove {
		t.Errorf("expected APPROVE, got %v", decision.Outcome)
	}
	if len(decision.Reasons) != 0 {
		t.Errorf("expected no reasons, got %d", len(decision.Reasons))
	}
	if len(trace.Evidence) != 1 {
		t.Errorf("expected 1 evidence, got %d", len(trace.Evidence))
	}
	if decision.PolicyID != "test-policy" {
		t.Errorf("expected policy ID test-policy, got %s", decision.PolicyID)
	}
	if decision.PolicyVersion != 1 {
		t.Errorf("expected policy version 1, got %d", decision.PolicyVersion)
	}
	if len(trace.Rules) != 1 {
		t.Errorf("expected 1 rule in trace, got %d", len(trace.Rules))
	}
	if trace.Rules[0].Result.Matched {
		t.Error("expected rule not to match")
	}
	if trace.Rules[0].Result.Actual != 30 {
		t.Errorf("expected actual value 30, got %v", trace.Rules[0].Result.Actual)
	}
	if trace.Rules[0].Result.Threshold != 18 {
		t.Errorf("expected threshold 18, got %v", trace.Rules[0].Result.Threshold)
	}
}

func TestEvaluate_Review(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Alice",
			Age:  30,
		},
		Application: &domain.Application{
			ID:              "app1",
			RequestedAmount: ptr(int64(150000)),
			Purpose:         "personal",
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "HIGH_AMOUNT",
				Description: "Requests over 100000 need review",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Application == nil || a.Application.RequestedAmount == nil {
						return domain.ConditionResult{Matched: false, Detail: "no application or requested amount"}
					}
					return domain.ConditionResult{
						Actual:    *a.Application.RequestedAmount,
						Threshold: int64(100000),
						Matched:   *a.Application.RequestedAmount > 100000,
						Detail:    "requested_amount > 100000",
					}
				},
				Outcome:    domain.RuleOutcomeReview,
				ReasonCode: "HIGH_REQUEST_AMOUNT",
				ReasonDesc: "Requested amount exceeds review threshold",
			},
		},
	}

	decision, _, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionReview {
		t.Errorf("expected REVIEW, got %v", decision.Outcome)
	}
	if len(decision.Reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d", len(decision.Reasons))
	}
	if decision.Reasons[0].Code != "HIGH_REQUEST_AMOUNT" {
		t.Errorf("expected reason code HIGH_REQUEST_AMOUNT, got %s", decision.Reasons[0].Code)
	}
}

func TestEvaluate_RejectByRule(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Bob",
			Age:  16,
		},
		Application: &domain.Application{
			ID:              "app1",
			RequestedAmount: ptr(int64(50000)),
			Purpose:         "personal",
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "AGE_MINIMUM",
				Description: "Applicant must be at least 18",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 18,
						Matched:   a.Applicant.Age < 18,
						Detail:    "age < 18",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Applicant is under 18",
			},
		},
	}

	decision, _, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionReject {
		t.Errorf("expected REJECT, got %v", decision.Outcome)
	}
	if len(decision.Reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d", len(decision.Reasons))
	}
	if decision.Reasons[0].Code != "AGE_BELOW_MINIMUM" {
		t.Errorf("expected reason code AGE_BELOW_MINIMUM, got %s", decision.Reasons[0].Code)
	}
}

func TestEvaluate_Knockout(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Charlie",
			Age:  25,
		},
		Application: &domain.Application{
			ID:              "app1",
			RequestedAmount: ptr(int64(50000)),
			Purpose:         "personal",
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Knockouts: []domain.Knockout{
			{
				Code:        "IDENTITY_FAILED",
				Description: "Identity verification failed",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:  a.Applicant.Name,
						Matched: a.Applicant.Name == "Charlie",
						Detail:  "name == Charlie",
					}
				},
				ReasonCode: "IDENTITY_VERIFICATION_FAILED",
				ReasonDesc: "Identity verification failed for applicant",
			},
		},
		Rules: []domain.Rule{
			{
				Code:        "AGE_MINIMUM",
				Description: "Applicant must be at least 18",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 18,
						Matched:   a.Applicant.Age < 18,
						Detail:    "age < 18",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Applicant is under 18",
			},
		},
	}

	decision, trace, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionReject {
		t.Errorf("expected REJECT from knockout, got %v", decision.Outcome)
	}
	if len(decision.Reasons) != 1 {
		t.Fatalf("expected 1 reason from knockout, got %d", len(decision.Reasons))
	}
	if decision.Reasons[0].Code != "IDENTITY_VERIFICATION_FAILED" {
		t.Errorf("expected knockout reason, got %s", decision.Reasons[0].Code)
	}
	if len(trace.Knockouts) != 1 {
		t.Errorf("expected 1 knockout in trace, got %d", len(trace.Knockouts))
	}
	if len(trace.Rules) != 1 {
		t.Errorf("expected 1 rule in trace, got %d", len(trace.Rules))
	}
	if len(trace.Evidence) != 2 {
		t.Errorf("expected 2 evidence entries, got %d", len(trace.Evidence))
	}
}

func TestEvaluate_MultipleKnockouts(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Ivan",
			Age:  16,
		},
		Application: &domain.Application{
			ID:              "app1",
			RequestedAmount: ptr(int64(50000)),
			Purpose:         "personal",
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Knockouts: []domain.Knockout{
			{
				Code:        "AGE_BELOW_MINIMUM",
				Description: "Applicant is under 18",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 18,
						Matched:   a.Applicant.Age < 18,
						Detail:    "age < 18",
					}
				},
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Applicant is under 18",
			},
			{
				Code:        "DSR_ABOVE_MAXIMUM",
				Description: "Debt service ratio too high",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:  0.95,
						Matched: true,
						Detail:  "dsr > 0.80",
					}
				},
				ReasonCode: "HIGH_DSR",
				ReasonDesc: "DSR exceeds maximum",
			},
		},
	}

	decision, trace, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionReject {
		t.Errorf("expected REJECT, got %v", decision.Outcome)
	}
	if len(trace.Knockouts) != 2 {
		t.Errorf("expected 2 knockouts in trace, got %d", len(trace.Knockouts))
	}
	if len(decision.Reasons) != 1 {
		t.Errorf("expected 1 reason (first knockout), got %d", len(decision.Reasons))
	}
	if decision.Reasons[0].Code != "AGE_BELOW_MINIMUM" {
		t.Errorf("expected first knockout reason, got %s", decision.Reasons[0].Code)
	}
}

func TestEvaluate_MultipleRules(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Diana",
			Age:  16,
		},
		Application: &domain.Application{
			ID:              "app1",
			RequestedAmount: ptr(int64(200000)),
			Purpose:         "personal",
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "AGE_MINIMUM",
				Description: "Applicant must be at least 18",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 18,
						Matched:   a.Applicant.Age < 18,
						Detail:    "age < 18",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Applicant is under 18",
			},
			{
				Code:        "HIGH_AMOUNT",
				Description: "Requests over 100000 need review",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Application == nil || a.Application.RequestedAmount == nil {
						return domain.ConditionResult{Matched: false, Detail: "no application or requested amount"}
					}
					return domain.ConditionResult{
						Actual:    *a.Application.RequestedAmount,
						Threshold: int64(100000),
						Matched:   *a.Application.RequestedAmount > 100000,
						Detail:    "requested_amount > 100000",
					}
				},
				Outcome:    domain.RuleOutcomeReview,
				ReasonCode: "HIGH_REQUEST_AMOUNT",
				ReasonDesc: "Requested amount exceeds review threshold",
			},
		},
	}

	decision, trace, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionReject {
		t.Errorf("expected REJECT, got %v", decision.Outcome)
	}
	if len(decision.Reasons) != 2 {
		t.Errorf("expected 2 reasons, got %d", len(decision.Reasons))
	}
	if len(trace.Evidence) != 2 {
		t.Errorf("expected 2 evidence entries, got %d", len(trace.Evidence))
	}
}

func TestEvaluate_Deterministic(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Eve",
			Age:  25,
		},
		Application: &domain.Application{
			ID:              "app1",
			RequestedAmount: ptr(int64(50000)),
			Purpose:         "personal",
		},
		Score: &domain.CreditScore{
			Value:    700,
			Provider: "mock",
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "SCORE_CHECK",
				Description: "Evaluate credit score",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Score == nil {
						return domain.ConditionResult{Matched: false, Detail: "no score"}
					}
					return domain.ConditionResult{
						Actual:    a.Score.Value,
						Threshold: 650,
						Matched:   a.Score.Value < 650,
						Detail:    "score < 650",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "LOW_CREDIT_SCORE",
				ReasonDesc: "Credit score below threshold",
			},
		},
		ScoreThresholds: &domain.ScoreThresholds{
			ApproveThreshold: 650,
			ReviewThreshold:  500,
		},
	}

	var first domain.Decision
	var firstTrace domain.EvaluationTrace
	for i := 0; i < 100; i++ {
		d, tr, err := Evaluate(assessment, policy)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if i == 0 {
			first = d
			firstTrace = tr
			continue
		}
		if d.Outcome != first.Outcome {
			t.Errorf("iteration %d: outcome changed from %v to %v", i, first.Outcome, d.Outcome)
		}
		if len(d.Reasons) != len(first.Reasons) {
			t.Errorf("iteration %d: reasons count changed from %d to %d", i, len(first.Reasons), len(d.Reasons))
		}
		if len(tr.Evidence) != len(firstTrace.Evidence) {
			t.Errorf("iteration %d: evidence count changed from %d to %d", i, len(firstTrace.Evidence), len(tr.Evidence))
		}
	}
}

func TestEvaluate_PolicyVersionDifferences(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Frank",
			Age:  20,
		},
		Application: &domain.Application{
			ID:              "app1",
			RequestedAmount: ptr(int64(80000)),
			Purpose:         "personal",
		},
	}

	policyV1 := domain.Policy{
		ID:      "loan-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "AMOUNT_REVIEW",
				Description: "Requests over 50000 need review",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Application == nil || a.Application.RequestedAmount == nil {
						return domain.ConditionResult{Matched: false, Detail: "no application or requested amount"}
					}
					return domain.ConditionResult{
						Actual:    *a.Application.RequestedAmount,
						Threshold: int64(50000),
						Matched:   *a.Application.RequestedAmount > 50000,
						Detail:    "requested_amount > 50000",
					}
				},
				Outcome:    domain.RuleOutcomeReview,
				ReasonCode: "HIGH_AMOUNT",
				ReasonDesc: "Amount exceeds threshold",
			},
		},
	}

	policyV2 := domain.Policy{
		ID:      "loan-policy",
		Version: 2,
		Rules: []domain.Rule{
			{
				Code:        "AMOUNT_REVIEW",
				Description: "Requests over 100000 need review",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Application == nil || a.Application.RequestedAmount == nil {
						return domain.ConditionResult{Matched: false, Detail: "no application or requested amount"}
					}
					return domain.ConditionResult{
						Actual:    *a.Application.RequestedAmount,
						Threshold: int64(100000),
						Matched:   *a.Application.RequestedAmount > 100000,
						Detail:    "requested_amount > 100000",
					}
				},
				Outcome:    domain.RuleOutcomeReview,
				ReasonCode: "HIGH_AMOUNT",
				ReasonDesc: "Amount exceeds threshold",
			},
		},
	}

	d1, _, err := Evaluate(assessment, policyV1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	d2, _, err := Evaluate(assessment, policyV2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if d1.Outcome == d2.Outcome {
		t.Errorf("expected different outcomes for different policy versions, both got %v", d1.Outcome)
	}
	if d1.PolicyVersion != 1 {
		t.Errorf("expected policy version 1, got %d", d1.PolicyVersion)
	}
	if d2.PolicyVersion != 2 {
		t.Errorf("expected policy version 2, got %d", d2.PolicyVersion)
	}
}

func TestEvaluate_EvidenceReferences(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Grace",
			Age:  16,
		},
		Application: &domain.Application{
			ID:              "app1",
			RequestedAmount: ptr(int64(50000)),
			Purpose:         "personal",
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "AGE_MINIMUM",
				Description: "Applicant must be at least 18",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 18,
						Matched:   a.Applicant.Age < 18,
						Detail:    "age < 18",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Applicant is under 18",
			},
		},
	}

	decision, trace, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(trace.Evidence) != 1 {
		t.Fatalf("expected 1 evidence, got %d", len(trace.Evidence))
	}
	if len(decision.Reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d", len(decision.Reasons))
	}

	evidence := trace.Evidence[0]
	reason := decision.Reasons[0]

	if evidence.Reference != "AGE_MINIMUM" {
		t.Errorf("expected evidence reference AGE_MINIMUM, got %s", evidence.Reference)
	}
	if reason.EvidenceRef != evidence.Reference {
		t.Errorf("reason evidence ref %s does not match evidence reference %s", reason.EvidenceRef, evidence.Reference)
	}
	if evidence.Source != "rule" {
		t.Errorf("expected evidence source rule, got %s", evidence.Source)
	}
	if reason.Value != 16 {
		t.Errorf("expected reason value 16, got %v", reason.Value)
	}
	if reason.Threshold != 18 {
		t.Errorf("expected reason threshold 18, got %v", reason.Threshold)
	}
}

func TestEvaluate_Explainability(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Hank",
			Age:  25,
		},
		Application: &domain.Application{
			ID:              "app1",
			RequestedAmount: ptr(int64(150000)),
			Purpose:         "personal",
		},
		Score: &domain.CreditScore{
			Value:    620,
			Provider: "mock",
		},
	}

	policy := domain.Policy{
		ID:      "personal-loan",
		Version: 3,
		Rules: []domain.Rule{
			{
				Code:        "HIGH_AMOUNT",
				Description: "Requests over 100000 need review",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Application == nil || a.Application.RequestedAmount == nil {
						return domain.ConditionResult{Matched: false, Detail: "no application or requested amount"}
					}
					return domain.ConditionResult{
						Actual:    *a.Application.RequestedAmount,
						Threshold: int64(100000),
						Matched:   *a.Application.RequestedAmount > 100000,
						Detail:    "requested_amount > 100000",
					}
				},
				Outcome:    domain.RuleOutcomeReview,
				ReasonCode: "HIGH_REQUEST_AMOUNT",
				ReasonDesc: "Requested amount exceeds review threshold",
			},
			{
				Code:        "LOW_SCORE",
				Description: "Scores below 650 are rejected",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Score == nil {
						return domain.ConditionResult{Matched: false, Detail: "no score"}
					}
					return domain.ConditionResult{
						Actual:    a.Score.Value,
						Threshold: 650,
						Matched:   a.Score.Value < 650,
						Detail:    "score < 650",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "LOW_CREDIT_SCORE",
				ReasonDesc: "Credit score below policy threshold",
			},
		},
	}

	decision, _, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionReject {
		t.Errorf("expected REJECT, got %v", decision.Outcome)
	}

	if len(decision.Reasons) != 2 {
		t.Fatalf("expected 2 reasons, got %d", len(decision.Reasons))
	}

	reasonCodes := make(map[string]bool)
	for _, r := range decision.Reasons {
		reasonCodes[r.Code] = true
		if r.Description == "" {
			t.Errorf("reason %s has empty description", r.Code)
		}
		if r.EvidenceRef == "" {
			t.Errorf("reason %s has empty evidence ref", r.Code)
		}
	}
	if !reasonCodes["HIGH_REQUEST_AMOUNT"] {
		t.Error("missing HIGH_REQUEST_AMOUNT reason")
	}
	if !reasonCodes["LOW_CREDIT_SCORE"] {
		t.Error("missing LOW_CREDIT_SCORE reason")
	}

	if decision.PolicyID != "personal-loan" {
		t.Errorf("expected policy ID personal-loan, got %s", decision.PolicyID)
	}
	if decision.PolicyVersion != 3 {
		t.Errorf("expected policy version 3, got %d", decision.PolicyVersion)
	}
}

func TestEvaluate_ScoreThresholds(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		expected domain.DecisionOutcome
	}{
		{"high score approves", 700, domain.DecisionApprove},
		{"medium score reviews", 600, domain.DecisionReview},
		{"low score rejects", 400, domain.DecisionReject},
		{"exactly at approve threshold", 650, domain.DecisionApprove},
		{"exactly at review threshold", 500, domain.DecisionReview},
		{"just below review threshold", 499, domain.DecisionReject},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment := domain.Assessment{
				ID: "a1",
				Applicant: domain.Applicant{
					ID:   "app1",
					Name: "Test",
					Age:  30,
				},
				Application: &domain.Application{
					ID:              "app1",
					RequestedAmount: ptr(int64(50000)),
					Purpose:         "personal",
				},
				Score: &domain.CreditScore{
					Value:    tt.score,
					Provider: "mock",
				},
			}

			policy := domain.Policy{
				ID:      "score-policy",
				Version: 1,
				ScoreThresholds: &domain.ScoreThresholds{
					ApproveThreshold: 650,
					ReviewThreshold:  500,
				},
			}

			decision, _, err := Evaluate(assessment, policy)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if decision.Outcome != tt.expected {
				t.Errorf("score %d: expected %v, got %v", tt.score, tt.expected, decision.Outcome)
			}
		})
	}
}

func TestEvaluate_NoScoreWithThresholds(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Test",
			Age:  30,
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		ScoreThresholds: &domain.ScoreThresholds{
			ApproveThreshold: 650,
			ReviewThreshold:  500,
		},
	}

	decision, _, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionApprove {
		t.Errorf("expected APPROVE, got %v", decision.Outcome)
	}
}

func TestEvaluate_DefaultOutcome(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Test",
			Age:  30,
		},
	}

	policy := domain.Policy{
		ID:             "test-policy",
		Version:        1,
		DefaultOutcome: domain.DecisionReview,
	}

	decision, _, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionReview {
		t.Errorf("expected REVIEW (default), got %v", decision.Outcome)
	}
}

func TestEvaluate_DefaultOutcomeNotUsedWhenRulesExist(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Test",
			Age:  30,
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "AGE_MINIMUM",
				Description: "Applicant must be at least 18",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 18,
						Matched:   a.Applicant.Age < 18,
						Detail:    "age < 18",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Applicant is under 18",
			},
		},
		DefaultOutcome: domain.DecisionReview,
	}

	decision, _, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionApprove {
		t.Errorf("expected APPROVE, got %v", decision.Outcome)
	}
}

func TestEvaluate_KnockoutTakesPrecedenceOverScore(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Knocked",
			Age:  30,
		},
		Score: &domain.CreditScore{
			Value:    800,
			Provider: "mock",
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Knockouts: []domain.Knockout{
			{
				Code:        "BLACKLISTED",
				Description: "Applicant is blacklisted",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:  a.Applicant.Name,
						Matched: a.Applicant.Name == "Knocked",
						Detail:  "name == Knocked",
					}
				},
				ReasonCode: "APPLICANT_BLACKLISTED",
				ReasonDesc: "Applicant is on the blacklist",
			},
		},
		ScoreThresholds: &domain.ScoreThresholds{
			ApproveThreshold: 650,
			ReviewThreshold:  500,
		},
	}

	decision, _, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionReject {
		t.Errorf("expected REJECT from knockout, got %v", decision.Outcome)
	}
}

func TestEvaluate_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		assessment  domain.Assessment
		errContains string
	}{
		{
			name:        "missing assessment ID",
			assessment:  domain.Assessment{Applicant: domain.Applicant{ID: "a"}},
			errContains: "assessment ID",
		},
		{
			name:        "missing applicant ID",
			assessment:  domain.Assessment{ID: "1"},
			errContains: "applicant ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := domain.Policy{ID: "p", Version: 1}
			_, _, err := Evaluate(tt.assessment, policy)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.errContains) {
				t.Errorf("expected error containing %q, got %v", tt.errContains, err)
			}
		})
	}
}

func TestEvaluate_RuleDoesNotMatch(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Test",
			Age:  30,
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "AGE_MINIMUM",
				Description: "Applicant must be at least 18",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 18,
						Matched:   a.Applicant.Age < 18,
						Detail:    "age < 18",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Applicant is under 18",
			},
		},
	}

	decision, trace, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Outcome != domain.DecisionApprove {
		t.Errorf("expected APPROVE, got %v", decision.Outcome)
	}
	if len(decision.Reasons) != 0 {
		t.Errorf("expected no reasons, got %d", len(decision.Reasons))
	}
	if len(trace.Evidence) != 1 {
		t.Errorf("expected 1 evidence, got %d", len(trace.Evidence))
	}
	if len(trace.Rules) != 1 {
		t.Errorf("expected 1 rule in trace, got %d", len(trace.Rules))
	}
	if trace.Rules[0].Result.Matched {
		t.Error("expected rule not to match in trace")
	}
}

func TestEvaluate_ConditionResultMetadata(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Test",
			Age:  25,
		},
		Application: &domain.Application{
			ID:              "app1",
			RequestedAmount: ptr(int64(75000)),
			Purpose:         "personal",
		},
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "HIGH_AMOUNT",
				Description: "Requests over 100000 need review",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Application == nil || a.Application.RequestedAmount == nil {
						return domain.ConditionResult{Matched: false, Detail: "no application or requested amount"}
					}
					return domain.ConditionResult{
						Actual:    *a.Application.RequestedAmount,
						Threshold: int64(100000),
						Matched:   *a.Application.RequestedAmount > 100000,
						Detail:    "requested_amount > 100000",
					}
				},
				Outcome:    domain.RuleOutcomeReview,
				ReasonCode: "HIGH_REQUEST_AMOUNT",
				ReasonDesc: "Requested amount exceeds review threshold",
			},
		},
	}

	_, trace, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(trace.Rules) != 1 {
		t.Fatalf("expected 1 rule in trace, got %d", len(trace.Rules))
	}

	r := trace.Rules[0]
	if r.Result.Actual != int64(75000) {
		t.Errorf("expected actual 75000, got %v", r.Result.Actual)
	}
	if r.Result.Threshold != int64(100000) {
		t.Errorf("expected threshold 100000, got %v", r.Result.Threshold)
	}
	if r.Result.Matched {
		t.Error("expected not matched")
	}
	if r.Result.Detail != "requested_amount > 100000" {
		t.Errorf("expected detail 'requested_amount > 100000', got %q", r.Result.Detail)
	}
}

func TestDecisionOutcomeString(t *testing.T) {
	tests := []struct {
		outcome domain.DecisionOutcome
		want    string
	}{
		{domain.DecisionApprove, "APPROVE"},
		{domain.DecisionReview, "REVIEW"},
		{domain.DecisionReject, "REJECT"},
		{domain.DecisionOutcome(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.outcome.String(); got != tt.want {
			t.Errorf("DecisionOutcome(%d).String() = %q, want %q", tt.outcome, got, tt.want)
		}
	}
}

func TestEvaluate_NoApplication(t *testing.T) {
	assessment := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app1",
			Name: "Test",
			Age:  30,
		},
		// No Application
	}

	policy := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "HIGH_AMOUNT",
				Description: "Requests over 100000 need review",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Application == nil || a.Application.RequestedAmount == nil {
						return domain.ConditionResult{Matched: false, Detail: "no application or requested amount"}
					}
					return domain.ConditionResult{
						Actual:    *a.Application.RequestedAmount,
						Threshold: int64(100000),
						Matched:   *a.Application.RequestedAmount > 100000,
						Detail:    "requested_amount > 100000",
					}
				},
				Outcome:    domain.RuleOutcomeReview,
				ReasonCode: "HIGH_REQUEST_AMOUNT",
				ReasonDesc: "Requested amount exceeds review threshold",
			},
		},
	}

	decision, _, err := Evaluate(assessment, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No application → rule doesn't match → APPROVE
	if decision.Outcome != domain.DecisionApprove {
		t.Errorf("expected APPROVE, got %v", decision.Outcome)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
