package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"groundclearance/internal/constants"
	"groundclearance/internal/model"
	"groundclearance/internal/util"
)

func TestClearanceTransition(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{constants.ClearancePending, constants.ClearanceCleared, true},
		{constants.ClearancePending, constants.ClearanceRestricted, true},
		{constants.ClearancePending, constants.ClearanceRevoked, true},
		{constants.ClearanceCleared, constants.ClearanceRevoked, true},
		{constants.ClearanceRestricted, constants.ClearanceRevoked, true},
		{constants.ClearanceRevoked, constants.ClearanceCleared, false},
		{constants.ClearanceCleared, constants.ClearanceRestricted, false},
	}
	for _, item := range cases {
		if got := allowedClearanceTransition(item.from, item.to); got != item.want {
			t.Fatalf("transition %s -> %s: got %v want %v", item.from, item.to, got, item.want)
		}
	}
}

func TestTurnaroundTransitionRedLines(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{constants.TurnaroundOpen, constants.TurnaroundChecking, true},
		{constants.TurnaroundDecisioned, constants.TurnaroundCompleted, true},
		{constants.TurnaroundOpen, constants.TurnaroundDecisioned, false},
		{constants.TurnaroundChecking, constants.TurnaroundCompleted, false},
		{constants.TurnaroundCompleted, constants.TurnaroundOpen, false},
	}
	for _, item := range cases {
		if got := allowedTurnaroundTransition(item.from, item.to); got != item.want {
			t.Fatalf("turnaround transition %s -> %s: got %v want %v", item.from, item.to, got, item.want)
		}
	}
}

func TestGroundUnitTransitionRedLines(t *testing.T) {
	if allowedUnitTransition(constants.UnitRetired, constants.UnitAvailable) {
		t.Fatal("retired equipment must be terminal")
	}
	if allowedUnitTransition(constants.UnitAvailable, constants.UnitAvailable) {
		t.Fatal("same-state equipment transition must be rejected")
	}
	if !allowedUnitTransition(constants.UnitBlocked, constants.UnitAvailable) {
		t.Fatal("authorized recovery from blocked state must remain possible")
	}
}

func TestNormalizeEvidence(t *testing.T) {
	items, err := normalizeEvidence([]string{" inspection-1.jpg ", "inspection-1.jpg", "meter-2.json"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0] != "inspection-1.jpg" {
		t.Fatalf("unexpected normalized evidence: %#v", items)
	}
	if _, err := normalizeEvidence([]string{"   "}); err == nil {
		t.Fatal("blank evidence must be rejected")
	} else {
		var appErr *util.AppError
		if !errors.As(err, &appErr) || appErr.Code != constants.CodeValidationFailed {
			t.Fatalf("unexpected blank evidence error: %v", err)
		}
	}
}

func TestSharedEnums(t *testing.T) {
	if !constants.IsValidUnitState(constants.UnitInspection) || constants.IsValidUnitState("broken") {
		t.Fatal("unit state validation mismatch")
	}
	if !constants.IsValidRiskLevel(constants.RiskCritical) || constants.IsValidRiskLevel("urgent") {
		t.Fatal("risk validation mismatch")
	}
}

func TestPlannedWindowOverlap(t *testing.T) {
	base := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	ninety := time.Duration(constants.ScheduleWindowMinutes) * time.Minute
	cases := []struct {
		name           string
		startA, startB time.Time
		want           bool
	}{
		{"fully disjoint", base, base.Add(3 * time.Hour), false},
		{"back to back chains", base, base.Add(ninety), false},
		{"one minute overlap", base, base.Add(ninety - time.Minute), true},
		{"surrounding window", base.Add(-30 * time.Minute), base, true},
	}
	for _, item := range cases {
		got := windowsOverlap(item.startA, item.startA.Add(ninety), item.startB, item.startB.Add(ninety))
		if got != item.want {
			t.Fatalf("%s: got %v want %v", item.name, got, item.want)
		}
	}
}

func TestTurnaroundHoldsWindow(t *testing.T) {
	if !turnaroundHolds(nil) {
		t.Fatal("missing decision is treated conservatively as still holding")
	}
	if !turnaroundHolds(&model.ClearanceDecision{State: constants.ClearancePending}) {
		t.Fatal("pending turnaround must still occupy its planned window")
	}
	if !turnaroundHolds(&model.ClearanceDecision{State: constants.ClearanceCleared}) {
		t.Fatal("cleared turnaround still occupies the schedule until completion")
	}
	if turnaroundHolds(&model.ClearanceDecision{State: constants.ClearanceRevoked}) {
		t.Fatal("revoked turnaround must release the planned window")
	}
}

func TestWindowConflictErrorNamesFlight(t *testing.T) {
	base := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	units := map[uint64]*model.GroundUnit{7: {ID: 7, UnitCode: "TUG-017"}}
	blocker := model.Turnaround{FlightNo: "CA1831", ScheduledAt: base}
	err := NewWindowConflictError(units, 7, blocker)
	if err == nil || err.Code != constants.CodeStateConflict {
		t.Fatalf("unexpected conflict error: %v", err)
	}
	for _, fragment := range []string{"CA1831", "TUG-017", "10:00", "11:30"} {
		if !strings.Contains(err.Message, fragment) {
			t.Fatalf("conflict message must contain %q, got %q", fragment, err.Message)
		}
	}
}
