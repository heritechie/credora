package policy

import (
	"testing"

	"credora/internal/domain"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	pol := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code: "RULE_1",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{Matched: false}
				},
				Outcome: domain.RuleOutcomePass,
			},
		},
	}

	if err := reg.Register(pol); err != nil {
		t.Fatalf("register: %v", err)
	}

	got, ok := reg.Get("test-policy", 1)
	if !ok {
		t.Fatal("expected policy to be found")
	}
	if got.ID != "test-policy" {
		t.Errorf("expected ID test-policy, got %s", got.ID)
	}
	if got.Version != 1 {
		t.Errorf("expected version 1, got %d", got.Version)
	}
	if len(got.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(got.Rules))
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	reg := NewRegistry()

	_, ok := reg.Get("nonexistent", 1)
	if ok {
		t.Error("expected policy not to be found")
	}
}

func TestRegistry_Has(t *testing.T) {
	reg := NewRegistry()

	if reg.Has("test-policy", 1) {
		t.Error("expected false for nonexistent policy")
	}

	pol := domain.Policy{ID: "test-policy", Version: 1}
	_ = reg.Register(pol)

	if !reg.Has("test-policy", 1) {
		t.Error("expected true for registered policy")
	}
	if reg.Has("test-policy", 2) {
		t.Error("expected false for unregistered version")
	}
}

func TestRegistry_DuplicateRegister(t *testing.T) {
	reg := NewRegistry()

	pol := domain.Policy{ID: "test-policy", Version: 1}
	if err := reg.Register(pol); err != nil {
		t.Fatalf("first register: %v", err)
	}

	err := reg.Register(pol)
	if err == nil {
		t.Fatal("expected error for duplicate register")
	}
}

func TestRegistry_MultiplePolicies(t *testing.T) {
	reg := NewRegistry()

	pol1 := domain.Policy{ID: "policy-a", Version: 1}
	pol2 := domain.Policy{ID: "policy-a", Version: 2}
	pol3 := domain.Policy{ID: "policy-b", Version: 1}

	_ = reg.Register(pol1)
	_ = reg.Register(pol2)
	_ = reg.Register(pol3)

	if !reg.Has("policy-a", 1) {
		t.Error("expected policy-a v1")
	}
	if !reg.Has("policy-a", 2) {
		t.Error("expected policy-a v2")
	}
	if !reg.Has("policy-b", 1) {
		t.Error("expected policy-b v1")
	}
	if reg.Has("policy-a", 3) {
		t.Error("expected no policy-a v3")
	}
}

func TestRegistry_Defaults(t *testing.T) {
	reg := NewRegistry()
	if err := RegisterDefaults(reg); err != nil {
		t.Fatalf("register defaults: %v", err)
	}

	pol, ok := reg.Get("personal-loan", 1)
	if !ok {
		t.Fatal("expected personal-loan v1 to be registered")
	}
	if pol.Name != "Personal Loan" {
		t.Errorf("expected name Personal Loan, got %s", pol.Name)
	}
	if pol.Version != 1 {
		t.Errorf("expected version 1, got %d", pol.Version)
	}
}
