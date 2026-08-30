// Package policy implements deterministic credit policy evaluation.
//
// Evaluation order:
//  1. Validate assessment input
//  2. Evaluate all knockout conditions
//  3. Evaluate all ordinary policy rules
//  4. Apply score thresholds if applicable
//  5. Apply precedence to determine final decision
//  6. Collect evidence from all evaluations
//  7. Produce structured decision with reasons and evaluation trace
//
// Precedence:
//   - Knockouts: highest priority, always produce REJECT
//   - Rules with REJECT outcome: produce REJECT
//   - Rules with REVIEW outcome: produce REVIEW
//   - Score thresholds: used when no knockouts or rules trigger
//   - DefaultOutcome from policy: used when nothing else determines the outcome
//
// The evaluator is deterministic: the same assessment input and policy version
// must always produce the same decision.
//
// All conditions are evaluated even when the outcome is already determined.
// This ensures the EvaluationTrace is complete for analysis and backtesting.
package policy

import (
	"fmt"
	"time"

	"credora/internal/domain"
)

// Evaluate performs deterministic policy evaluation against an assessment.
// It returns the final Decision, a complete EvaluationTrace for analysis,
// and any validation error.
func Evaluate(assessment domain.Assessment, policy domain.Policy) (domain.Decision, domain.EvaluationTrace, error) {
	if err := Validate(assessment); err != nil {
		return domain.Decision{}, domain.EvaluationTrace{}, fmt.Errorf("invalid assessment: %w", err)
	}

	now := time.Now()

	trace := domain.EvaluationTrace{
		Knockouts: make([]domain.KnockoutEvaluation, 0),
		Rules:     make([]domain.RuleEvaluation, 0),
		Evidence:  make([]domain.Evidence, 0),
	}

	// Phase 1: Evaluate all knockouts
	for _, ko := range policy.Knockouts {
		result := ko.Condition(assessment)
		trace.Knockouts = append(trace.Knockouts, domain.KnockoutEvaluation{
			Code:        ko.Code,
			Description: ko.Description,
			Result:      result,
			ReasonCode:  ko.ReasonCode,
			ReasonDesc:  ko.ReasonDesc,
		})
		trace.Evidence = append(trace.Evidence, domain.Evidence{
			Source:      "knockout",
			Field:       ko.Code,
			Value:       result.Matched,
			RetrievedAt: now,
			Reference:   ko.Code,
		})
	}

	// Phase 2: Evaluate all rules
	for _, r := range policy.Rules {
		result := r.Condition(assessment)
		trace.Rules = append(trace.Rules, domain.RuleEvaluation{
			Code:        r.Code,
			Description: r.Description,
			Result:      result,
			Outcome:     r.Outcome,
			ReasonCode:  r.ReasonCode,
			ReasonDesc:  r.ReasonDesc,
		})
		trace.Evidence = append(trace.Evidence, domain.Evidence{
			Source:      "rule",
			Field:       r.Code,
			Value:       result.Matched,
			RetrievedAt: now,
			Reference:   r.Code,
		})
	}

	// Phase 3: Apply precedence to determine outcome
	decision := applyPrecedence(policy, assessment, trace, now)

	return decision, trace, nil
}

// applyPrecedence determines the final decision from the evaluation trace.
func applyPrecedence(policy domain.Policy, assessment domain.Assessment, trace domain.EvaluationTrace, now time.Time) domain.Decision {
	decision := domain.Decision{
		PolicyID:      policy.ID,
		PolicyVersion: policy.Version,
		Reasons:       make([]domain.DecisionReason, 0),
	}

	// Check knockouts first
	for _, ko := range trace.Knockouts {
		if ko.Result.Matched {
			decision.Outcome = domain.DecisionReject
			decision.Reasons = append(decision.Reasons, domain.DecisionReason{
				Code:        ko.ReasonCode,
				Description: ko.ReasonDesc,
				Value:       ko.Result.Actual,
				Threshold:   ko.Result.Threshold,
				EvidenceRef: ko.Code,
			})
			return decision
		}
	}

	// Check rules
	mostSevere := domain.DecisionApprove
	for _, r := range trace.Rules {
		if r.Result.Matched {
			decision.Reasons = append(decision.Reasons, domain.DecisionReason{
				Code:        r.ReasonCode,
				Description: r.ReasonDesc,
				Value:       r.Result.Actual,
				Threshold:   r.Result.Threshold,
				EvidenceRef: r.Code,
			})

			ruleDecision := ruleOutcomeToDecision(r.Outcome)
			if ruleDecision > mostSevere {
				mostSevere = ruleDecision
			}
		}
	}

	// Score thresholds (if no knockouts or reject rules triggered)
	if mostSevere != domain.DecisionReject && policy.ScoreThresholds != nil && assessment.Score != nil {
		scoreDecision := scoreToDecision(assessment.Score.Value, *policy.ScoreThresholds)
		if scoreDecision > mostSevere {
			mostSevere = scoreDecision
		}
	}

	// Use policy default if no condition determined the outcome
	if mostSevere == domain.DecisionApprove && len(trace.Knockouts) == 0 && len(trace.Rules) == 0 {
		mostSevere = policy.DefaultOutcome
	}

	decision.Outcome = mostSevere
	return decision
}

// Validate checks that an assessment has the minimum required fields.
// An assessment requires an ID and an applicant. Application is optional:
// not all assessment types have an application context (e.g., limit assessment).
func Validate(a domain.Assessment) error {
	if a.ID == "" {
		return fmt.Errorf("assessment ID is required")
	}
	if a.Applicant.ID == "" {
		return fmt.Errorf("applicant ID is required")
	}
	return nil
}

func ruleOutcomeToDecision(o domain.RuleOutcome) domain.DecisionOutcome {
	switch o {
	case domain.RuleOutcomeReject:
		return domain.DecisionReject
	case domain.RuleOutcomeReview:
		return domain.DecisionReview
	default:
		return domain.DecisionApprove
	}
}

// scoreToDecision determines the decision from a score value and thresholds.
func scoreToDecision(score int, t domain.ScoreThresholds) domain.DecisionOutcome {
	if score >= t.ApproveThreshold {
		return domain.DecisionApprove
	}
	if score >= t.ReviewThreshold {
		return domain.DecisionReview
	}
	return domain.DecisionReject
}
