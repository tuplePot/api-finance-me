package domain

import (
	"testing"
	"time"

	"financial-tracker/internal/shared/apperror"
)

func TestNew(t *testing.T) {
	now := time.Date(2026, 11, 6, 3, 0, 0, 0, time.UTC)
	day := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)

	tx, err := New("id", "owner", Params{
		Type: Expense, Amount: 70_000, Category: "  Food & Drinks ", Date: day, Note: " yogurt ",
	}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.Category != "Food & Drinks" || tx.Note != "yogurt" {
		t.Errorf("fields not trimmed: %q / %q", tx.Category, tx.Note)
	}

	_, err = New("id", "owner", Params{Type: "gift", Amount: 0}, now)
	fields := apperror.FieldsOf(err)
	for _, f := range []string{"type", "amount", "category", "date"} {
		if fields[f] == "" {
			t.Errorf("expected validation error on %q, got %v", f, fields)
		}
	}
	if apperror.KindOf(err) != apperror.KindInvalid {
		t.Errorf("expected KindInvalid, got %v", apperror.KindOf(err))
	}
}
