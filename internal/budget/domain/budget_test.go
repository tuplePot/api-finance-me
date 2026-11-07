package domain

import (
	"testing"
	"time"

	"financial-tracker/internal/shared/apperror"
)

func TestNewAndUpdate(t *testing.T) {
	now := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)

	b, err := New("id", "owner", Params{Category: " Food & Drinks ", Period: Weekly, Limit: 350_000}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Category != "Food & Drinks" {
		t.Errorf("category not trimmed: %q", b.Category)
	}

	if err := b.Update("daily", 0, now); err == nil {
		t.Fatal("expected validation error")
	} else if f := apperror.FieldsOf(err); f["period"] == "" || f["limit"] == "" {
		t.Errorf("expected period+limit errors, got %v", f)
	}
	if b.Period != Weekly || b.Limit != 350_000 {
		t.Error("failed update must not change the budget")
	}

	if err := b.Update(Monthly, 1_500_000, now); err != nil {
		t.Fatal(err)
	}
	if b.Period != Monthly || b.Limit != 1_500_000 {
		t.Errorf("update not applied: %+v", b)
	}
}

func TestDuplicateMessage(t *testing.T) {
	if got := (DuplicateError{Category: "Snacks"}).Error(); got != "Snacks already has a budget" {
		t.Errorf("got %q", got)
	}
}
