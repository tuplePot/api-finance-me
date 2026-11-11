package domain

import (
	"testing"
	"time"

	"financial-tracker/internal/shared/apperror"
)

func TestNew(t *testing.T) {
	now := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)

	it, err := New("id", "owner", Params{Name: " Running shoes ", Price: 900_000}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if it.Name != "Running shoes" {
		t.Errorf("name not trimmed: %q", it.Name)
	}

	_, err = New("id", "owner", Params{Name: "  ", Price: 0}, now)
	if f := apperror.FieldsOf(err); f["name"] == "" || f["price"] == "" {
		t.Errorf("expected name+price errors, got %v", f)
	}
}
