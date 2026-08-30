package policy

import (
	"context"
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
func defaultPersonalLoanV1() domain.Policy {
	return domain.Policy{
		ID:          "personal-loan",
		Version:     1,
		Name:        "Personal Loan",
		Description: "Default personal loan assessment policy",
		Knockouts: []domain.Knockout{
			{
				Code:        "AGE_BELOW_MINIMUM",
				Description: "Applicant must be at least 18 years old",
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
		},
		Rules: []domain.Rule{
			{
				Code:        "HIGH_REQUEST_AMOUNT",
				Description: "Requests over 100000 require review",
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
				Code:        "LOW_CREDIT_SCORE",
				Description: "Credit score below 650 is rejected",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Score == nil {
						return domain.ConditionResult{Matched: false, Detail: "no score provided"}
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
				ReasonDesc: "Credit score is below the minimum threshold",
			},
		},
		ScoreThresholds: &domain.ScoreThresholds{
			ApproveThreshold: 700,
			ReviewThreshold:  500,
		},
		DefaultOutcome: domain.DecisionReview,
	}
}
