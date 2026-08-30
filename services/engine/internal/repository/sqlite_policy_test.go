package repository

import (
	"context"
	"testing"
)

func TestSQLitePolicyRepository_SaveAndGet(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLitePolicyRepository(db)
	ctx := context.Background()

	meta := PolicyMetadata{
		ID:          "personal-loan",
		Version:     1,
		Name:        "Personal Loan",
		Description: "Default personal loan policy",
	}

	if err := repo.Save(ctx, meta); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.Get(ctx, "personal-loan", 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.ID != "personal-loan" {
		t.Errorf("expected ID personal-loan, got %s", got.ID)
	}
	if got.Version != 1 {
		t.Errorf("expected version 1, got %d", got.Version)
	}
	if got.Name != "Personal Loan" {
		t.Errorf("expected name Personal Loan, got %s", got.Name)
	}
	if got.Description != "Default personal loan policy" {
		t.Errorf("expected description, got %s", got.Description)
	}
}

func TestSQLitePolicyRepository_GetNotFound(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLitePolicyRepository(db)
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent", 1)
	if err == nil {
		t.Fatal("expected error for nonexistent policy, got nil")
	}
}

func TestSQLitePolicyRepository_Exists(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLitePolicyRepository(db)
	ctx := context.Background()

	exists, err := repo.Exists(ctx, "personal-loan", 1)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists {
		t.Error("expected false for nonexistent policy")
	}

	meta := PolicyMetadata{
		ID:      "personal-loan",
		Version: 1,
		Name:    "Personal Loan",
	}
	if err := repo.Save(ctx, meta); err != nil {
		t.Fatalf("save: %v", err)
	}

	exists, err = repo.Exists(ctx, "personal-loan", 1)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !exists {
		t.Error("expected true for existing policy")
	}
}

func TestSQLitePolicyRepository_SaveOverwrite(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLitePolicyRepository(db)
	ctx := context.Background()

	meta1 := PolicyMetadata{
		ID:          "personal-loan",
		Version:     1,
		Name:        "Personal Loan v1",
		Description: "Original",
	}
	if err := repo.Save(ctx, meta1); err != nil {
		t.Fatalf("save: %v", err)
	}

	meta2 := PolicyMetadata{
		ID:          "personal-loan",
		Version:     1,
		Name:        "Personal Loan v1 Updated",
		Description: "Updated",
	}
	if err := repo.Save(ctx, meta2); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.Get(ctx, "personal-loan", 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Name != "Personal Loan v1 Updated" {
		t.Errorf("expected updated name, got %s", got.Name)
	}
	if got.Description != "Updated" {
		t.Errorf("expected updated description, got %s", got.Description)
	}
}

func TestSQLitePolicyRepository_MultipleVersions(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLitePolicyRepository(db)
	ctx := context.Background()

	meta1 := PolicyMetadata{ID: "loan", Version: 1, Name: "Loan v1"}
	meta2 := PolicyMetadata{ID: "loan", Version: 2, Name: "Loan v2"}

	if err := repo.Save(ctx, meta1); err != nil {
		t.Fatalf("save v1: %v", err)
	}
	if err := repo.Save(ctx, meta2); err != nil {
		t.Fatalf("save v2: %v", err)
	}

	got1, err := repo.Get(ctx, "loan", 1)
	if err != nil {
		t.Fatalf("get v1: %v", err)
	}
	got2, err := repo.Get(ctx, "loan", 2)
	if err != nil {
		t.Fatalf("get v2: %v", err)
	}

	if got1.Name != "Loan v1" {
		t.Errorf("expected v1 name, got %s", got1.Name)
	}
	if got2.Name != "Loan v2" {
		t.Errorf("expected v2 name, got %s", got2.Name)
	}
}
