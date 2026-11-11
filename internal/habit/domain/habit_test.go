package domain

import (
	"testing"
	"time"

	"financial-tracker/internal/shared/apperror"
)

func TestNewAndApply(t *testing.T) {
	now := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	h, err := New("id", "owner", Params{
		Name: " Cigarettes ", Icon: "lucide:cigarette", Cost: 35_000, Frequency: Daily, Since: since,
	}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Name != "Cigarettes" {
		t.Errorf("name not trimmed: %q", h.Name)
	}

	bad := Frequency("yearly")
	zero := int64(0)
	if err := h.Apply(Patch{Frequency: &bad, Cost: &zero}, now); err == nil {
		t.Fatal("expected validation error")
	} else if f := apperror.FieldsOf(err); f["frequency"] == "" || f["cost"] == "" {
		t.Errorf("expected frequency+cost errors, got %v", f)
	}
	if h.Frequency != Daily || h.Cost != 35_000 {
		t.Error("failed patch must not change the habit")
	}

	weekly := Weekly
	cost := int64(100_000)
	if err := h.Apply(Patch{Frequency: &weekly, Cost: &cost}, now); err != nil {
		t.Fatal(err)
	}
	if h.Frequency != Weekly || h.Cost != 100_000 || h.Name != "Cigarettes" {
		t.Errorf("patch not applied correctly: %+v", h)
	}
}

func TestNewRequiresFields(t *testing.T) {
	_, err := New("id", "owner", Params{}, time.Now())
	f := apperror.FieldsOf(err)
	for _, field := range []string{"name", "icon", "cost", "frequency", "since"} {
		if f[field] == "" {
			t.Errorf("expected %s error, got %v", field, f)
		}
	}
}
