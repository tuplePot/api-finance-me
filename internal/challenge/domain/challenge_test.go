package domain

import (
	"errors"
	"testing"
	"time"

	"financial-tracker/internal/shared/apperror"
)

var (
	now   = time.Date(2026, 11, 6, 5, 0, 0, 0, time.UTC)
	today = time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
)

func yogurt(t *testing.T) Challenge {
	t.Helper()
	c, err := New("c1", "owner", Params{
		Title:        "Skip the yogurt",
		Temptation:   Item{Name: "Yogurt", Price: 70_000},
		Goal:         Item{Name: "Dinner", Price: 35_000},
		TargetAmount: 140_000,
		Deadline:     today.AddDate(0, 0, 14),
	}, today, now)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewValidation(t *testing.T) {
	_, err := New("c1", "owner", Params{
		Temptation: Item{Price: -1},
		Deadline:   today.AddDate(0, 0, -1),
	}, today, now)
	if apperror.KindOf(err) != apperror.KindInvalid {
		t.Fatalf("expected invalid, got %v", err)
	}
	f := apperror.FieldsOf(err)
	for _, k := range []string{"title", "temptation.name", "temptation.price", "goal.name", "targetAmount", "deadline"} {
		if f[k] == "" {
			t.Errorf("missing error for %q (got %v)", k, f)
		}
	}
}

func TestRecordReachesTarget(t *testing.T) {
	c := yogurt(t)

	if _, done, err := c.Record("l1", Slip, today, now); err != nil || done {
		t.Fatalf("slip: done=%v err=%v", done, err)
	}
	if c.SavedAmount != 0 {
		t.Errorf("slip must not save money, saved=%d", c.SavedAmount)
	}

	if _, done, _ := c.Record("l2", Skip, today, now); done {
		t.Fatal("one skip (70K) must not complete a 140K target")
	}
	log, done, err := c.Record("l3", Skip, today, now)
	if err != nil || !done {
		t.Fatalf("second skip should complete: done=%v err=%v", done, err)
	}
	if log.Amount != 70_000 || c.SavedAmount != 140_000 || c.Status != Success || c.CompletedAt == nil {
		t.Errorf("unexpected state: %+v", c)
	}
	if len(c.Logs) != 3 || c.Logs[0].ID != "l3" {
		t.Errorf("logs must be newest first: %+v", c.Logs)
	}

	if _, _, err := c.Record("l4", Skip, today, now); !errors.Is(err, ErrClosed) {
		t.Errorf("closed challenge must reject logs, got %v", err)
	}
}

func TestDeadlineFailsChallenge(t *testing.T) {
	c := yogurt(t)
	lastDay := c.Deadline
	if c.Settle(lastDay) {
		t.Fatal("deadline day itself is still allowed")
	}
	after := lastDay.AddDate(0, 0, 1)
	if _, _, err := c.Record("l1", Skip, after, now); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed after deadline, got %v", err)
	}
	if c.Status != Failed {
		t.Errorf("expected failed, got %s", c.Status)
	}
}
