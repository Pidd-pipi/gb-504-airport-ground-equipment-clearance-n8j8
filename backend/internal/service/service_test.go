package service

import (
	"errors"
	"testing"
	"time"

	"groundclearance/internal/constants"
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

func TestOccupancyWindowOverlap(t *testing.T) {
	base := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	window := constants.OccupancyWindow
	cases := []struct {
		name string
		a, b time.Time
		want bool
	}{
		{"same start", base, base, true},
		{"partial overlap", base, base.Add(30 * time.Minute), true},
		{"overlap from earlier flight", base.Add(45 * time.Minute), base, true},
		{"back to back after", base, base.Add(window), false},
		{"back to back before", base.Add(window), base, false},
		{"disjoint", base, base.Add(2 * window), false},
	}
	for _, item := range cases {
		if got := windowsOverlap(item.a, item.b, window); got != item.want {
			t.Fatalf("%s: got %v want %v", item.name, got, item.want)
		}
	}
}

func TestTurnaroundOccupiesWindow(t *testing.T) {
	cases := []struct {
		status, clearance string
		want              bool
	}{
		{constants.TurnaroundOpen, "", true},
		{constants.TurnaroundChecking, "", true},
		{constants.TurnaroundDecisioned, constants.ClearanceCleared, true},
		{constants.TurnaroundDecisioned, constants.ClearanceRestricted, true},
		{constants.TurnaroundDecisioned, constants.ClearanceRevoked, false},
		{constants.TurnaroundCompleted, constants.ClearanceCleared, false},
	}
	for _, item := range cases {
		if got := occupiesWindow(item.status, item.clearance); got != item.want {
			t.Fatalf("occupies %s/%s: got %v want %v", item.status, item.clearance, got, item.want)
		}
	}
}
