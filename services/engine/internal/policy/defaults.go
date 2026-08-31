package policy

import (
	"context"
	"fmt"
	"log/slog"

	"credora/internal/domain"
	"credora/internal/repository"
)

// RegisterDefaults registers the built-in default policies in the registry.
// It returns an error if any policy fails to register.
func RegisterDefaults(reg *Registry) error {
	defaults := []domain.Policy{
		defaultPersonalLoanV1(),
	}

	for _, pol := range defaults {
		if err := reg.Register(pol); err != nil {
			return err
		}
	}
	return nil
}

// SeedPolicies persists policy metadata for all registered policies.
// It is idempotent: already-existing policies are silently skipped.
func SeedPolicies(ctx context.Context, reg *Registry, policyRepo repository.PolicyRepository, logger *slog.Logger) error {
	policies := []domain.Policy{
		defaultPersonalLoanV1(),
	}

	for _, pol := range policies {
		exists, err := policyRepo.Exists(ctx, pol.ID, pol.Version)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		meta := repository.PolicyMetadata{
			ID:          pol.ID,
			Version:     pol.Version,
			Name:        pol.Name,
			Description: pol.Description,
		}
		if err := policyRepo.Save(ctx, meta); err != nil {
			return err
		}
		logger.Info("seeded policy", "policy_id", pol.ID, "version", pol.Version)
	}
	return nil
}

// defaultPersonalLoanV1 returns the default personal loan policy v1.
// This demo policy supports both limit assessments (no application) and
// loan application assessments (with requested amount). It demonstrates:
// - Knockout conditions (age, DSR)
// - Credit score thresholds
// - Credit limit calculation (monthly income × 2)
// - Approved amount (min of requested amount and credit limit)
// - Requested amount exceeding limit behavior
// - Deterministic evaluation with full evidence
func defaultPersonalLoanV1() domain.Policy {
	// HIGH_DSR knockout: DSR = monthly_obligations / monthly_income > 70%
	// Computed without floating point: monthly_obligations * 100 > monthly_income * 70
	// Financial facts are optional inputs: when missing, the knockout does not trigger.
	highDSRKnockout := domain.Knockout{
		Code:        "HIGH_DSR",
		Description: "Debt service ratio exceeds policy threshold",
		Condition: func(a domain.Assessment) domain.ConditionResult {
			if a.MonthlyIncome == nil || a.MonthlyObligations == nil || *a.MonthlyIncome <= 0 {
				return domain.ConditionResult{Matched: false, Detail: "income or obligations not provided"}
			}
			matched := *a.MonthlyObligations*100 > *a.MonthlyIncome*70
			return domain.ConditionResult{
				Actual:    int64(*a.MonthlyObligations * 100),
				Threshold: int64(*a.MonthlyIncome * 70),
				Matched:   matched,
				Detail:    fmt.Sprintf("obligations*100 %d > income*70 %d", *a.MonthlyObligations*100, *a.MonthlyIncome*70),
			}
		},
		ReasonCode: "HIGH_DSR",
		ReasonDesc: "Debt service ratio exceeds policy threshold",
	}

	// Credit limit: MonthlyIncome × 2 (smallest currency unit)
	creditLimit := func(a domain.Assessment) int64 {
		if a.MonthlyIncome != nil && *a.MonthlyIncome > 0 {
			return *a.MonthlyIncome * 2
		}
		return 0
	}

	// Decision outputs:
	//   - Credit limit: MonthlyIncome × 2
	//   - Approved amount: min(requested_amount, credit_limit), only when a
	//     requested amount is present
	outputs := func(a domain.Assessment) *domain.DecisionOutputs {
		if a.MonthlyIncome == nil || *a.MonthlyIncome <= 0 {
			return nil
		}
		limit := *a.MonthlyIncome * 2
		out := &domain.DecisionOutputs{CreditLimit: &limit}
		if a.Application != nil && a.Application.RequestedAmount != nil {
			req := *a.Application.RequestedAmount
			approved := req
			if limit < approved {
				approved = limit
			}
			out.ApprovedAmount = &approved
		}
		return out
	}

	// Credit limit rule: if requested_amount > credit_limit → REVIEW
	requestedAmountExceedsLimit := func(a domain.Assessment) domain.ConditionResult {
		cl := creditLimit(a)
		var requestedAmount int64
		if a.Application != nil && a.Application.RequestedAmount != nil {
			requestedAmount = *a.Application.RequestedAmount
		}
		if requestedAmount > cl {
			return domain.ConditionResult{
				Actual:    requestedAmount,
				Threshold: cl,
				Matched:   true,
				Detail:    fmt.Sprintf("requested_amount %d > credit_limit %d", requestedAmount, cl),
			}
		}
		return domain.ConditionResult{Matched: false, Detail: "requested amount within credit limit"}
	}

	// CREDIT_SCORE_REVIEW rule: score between 600 and 699 → REVIEW
	creditScoreReview := domain.Rule{
		Code:        "CREDIT_SCORE_REVIEW",
		Description: "Credit score between 600 and 699 triggers review",
		Condition: func(a domain.Assessment) domain.ConditionResult {
			if a.Score == nil {
				return domain.ConditionResult{Matched: false, Detail: "no score provided"}
			}
			matched := a.Score.Value >= 600 && a.Score.Value <= 699
			return domain.ConditionResult{
				Actual:    a.Score.Value,
				Threshold: 650,
				Matched:   matched,
				Detail:    fmt.Sprintf("score %d in review range [600,699]", a.Score.Value),
			}
		},
		Outcome:    domain.RuleOutcomeReview,
		ReasonCode: "CREDIT_SCORE_REVIEW",
		ReasonDesc: "Credit score is between 600 and 699",
	}

	// CREDIT_SCORE_LOW rule: score < 600 → REJECT
	creditScoreLow := domain.Rule{
		Code:        "CREDIT_SCORE_LOW",
		Description: "Credit score below 600 is rejected",
		Condition: func(a domain.Assessment) domain.ConditionResult {
			if a.Score == nil {
				return domain.ConditionResult{Matched: false, Detail: "no score provided"}
			}
			matched := a.Score.Value < 600
			return domain.ConditionResult{
				Actual:    a.Score.Value,
				Threshold: 600,
				Matched:   matched,
				Detail:    fmt.Sprintf("score %d below 600", a.Score.Value),
			}
		},
		Outcome:    domain.RuleOutcomeReject,
		ReasonCode: "CREDIT_SCORE_LOW",
		ReasonDesc: "Credit score is below 600",
	}

	// HIGH_REQUEST_AMOUNT rule: requests over 100000 require review
	highRequestAmount := domain.Rule{
		Code:        "HIGH_REQUEST_AMOUNT",
		Description: "Requests over 100000 require review",
		Condition: func(a domain.Assessment) domain.ConditionResult {
			if a.Application == nil || a.Application.RequestedAmount == nil {
				return domain.ConditionResult{Matched: false, Detail: "no application or requested amount"}
			}
			matched := *a.Application.RequestedAmount > 100000
			return domain.ConditionResult{
				Actual:    *a.Application.RequestedAmount,
				Threshold: int64(100000),
				Matched:   matched,
				Detail:    "requested_amount > 100000",
			}
		},
		Outcome:    domain.RuleOutcomeReview,
		ReasonCode: "HIGH_REQUEST_AMOUNT",
		ReasonDesc: "Requested amount exceeds review threshold",
	}

	// LOW_CREDIT_SCORE rule: score < 650 → REJECT (existing behavior preserved)
	lowCreditScore := domain.Rule{
		Code:        "LOW_CREDIT_SCORE",
		Description: "Credit score below 650 is rejected",
		Condition: func(a domain.Assessment) domain.ConditionResult {
			if a.Score == nil {
				return domain.ConditionResult{Matched: false, Detail: "no score provided"}
			}
			matched := a.Score.Value < 650
			return domain.ConditionResult{
				Actual:    a.Score.Value,
				Threshold: 650,
				Matched:   matched,
				Detail:    "score < 650",
			}
		},
		Outcome:    domain.RuleOutcomeReject,
		ReasonCode: "LOW_CREDIT_SCORE",
		ReasonDesc: "Credit score is below the minimum threshold",
	}

	return domain.Policy{
		ID:          "personal-loan",
		Version:     1,
		Name:        "Personal Loan",
		Description: "Deterministic personal loan decision policy for demonstration and testing",
		Knockouts: []domain.Knockout{
			{
				Code:        "AGE_BELOW_MINIMUM",
				Description: "Applicant must be at least 21 years old",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 21,
						Matched:   a.Applicant.Age < 21,
						Detail:    "age < 21",
					}
				},
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Applicant is under 21",
			},
			highDSRKnockout,
		},
		Rules: []domain.Rule{
			creditScoreReview,
			creditScoreLow,
			highRequestAmount,
			lowCreditScore,
			{
				Code:        "REQUESTED_AMOUNT_EXCEEDS_LIMIT",
				Description: "Requested amount exceeds credit limit",
				Condition:   requestedAmountExceedsLimit,
				Outcome:     domain.RuleOutcomeReview,
				ReasonCode:  "REQUESTED_AMOUNT_EXCEEDS_LIMIT",
				ReasonDesc:  "Requested amount exceeds credit limit",
			},
		},
		ScoreThresholds: &domain.ScoreThresholds{
			ApproveThreshold: 700,
			ReviewThreshold:  600,
		},
		Outputs:         outputs,
		DefaultOutcome: domain.DecisionReview,
	}
}
