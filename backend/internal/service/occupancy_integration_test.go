//go:build integration

package service

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"groundclearance/internal/constants"
	"groundclearance/internal/model"
	"groundclearance/internal/repository"
	"groundclearance/internal/util"
)

// TestOccupancyIntegration exercises the schedule-window occupancy rules
// against a real PostgreSQL instance: 90-minute reservation windows,
// back-to-back scheduling, conflict messages naming the flight, and release
// through completion or clearance revocation.
func TestOccupancyIntegration(t *testing.T) {
	database := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(15432).
		StartParameters(map[string]string{"timezone": "UTC"}))
	if err := database.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	defer func() {
		if err := database.Stop(); err != nil {
			t.Fatalf("stop embedded postgres: %v", err)
		}
	}()
	dsn := "host=localhost port=15432 user=postgres password=postgres dbname=postgres sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect embedded postgres: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.GroundUnit{}, &model.Turnaround{},
		&model.SafetyCheck{}, &model.ClearanceDecision{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	logger := slog.Default()
	userRepo := repository.NewUserRepository(db)
	unitRepo := repository.NewGroundUnitRepository(db)
	turnaroundRepo := repository.NewTurnaroundRepository(db)
	checkRepo := repository.NewSafetyCheckRepository(db)
	clearanceRepo := repository.NewClearanceDecisionRepository(db)
	turnaroundSvc := NewTurnaroundService(db, turnaroundRepo, checkRepo, clearanceRepo, unitRepo, userRepo, logger)
	unitSvc := NewGroundUnitService(db, unitRepo, turnaroundRepo, clearanceRepo, logger)

	coordinator := &model.User{Phone: "13900000001", PasswordHash: "x", Name: "协调", Role: constants.RoleSafetyManager}
	if err := db.Create(coordinator).Error; err != nil {
		t.Fatalf("create coordinator: %v", err)
	}
	units := []model.GroundUnit{
		{UnitCode: "TUG-01", Name: "牵引车", UnitType: "tug", Stand: "A1", State: constants.UnitAvailable},
		{UnitCode: "GPU-01", Name: "电源车", UnitType: "gpu", Stand: "A1", State: constants.UnitAvailable},
	}
	if err := db.Create(&units).Error; err != nil {
		t.Fatalf("create units: %v", err)
	}
	actor := AuditContext{OperatorID: coordinator.ID, OperatorName: "协调"}
	base := time.Now().Add(2 * time.Hour).Truncate(time.Minute)
	unitIDs := []string{itoa(units[0].ID)}
	createTurnaround := func(flight string, scheduled time.Time, unitList []string) error {
		row := &model.Turnaround{FlightNo: flight, Stand: "A1", Phase: "servicing", ScheduledAt: scheduled,
			RiskLevel: constants.RiskMedium, GroundUnitIDs: model.JSONList(unitList), CoordinatorID: coordinator.ID}
		checks := []model.SafetyCheck{{CheckCode: "OPS-1", ItemName: "外观检查", RiskLevel: constants.RiskMedium}}
		_, err := turnaroundSvc.Create(row, checks, actor)
		return err
	}

	// 1. First turnaround occupies [base, base+90min).
	if err := createTurnaround("CA100", base, unitIDs); err != nil {
		t.Fatalf("first turnaround should be created: %v", err)
	}
	// 2. Overlapping window is rejected and names the conflicting flight.
	err = createTurnaround("CA200", base.Add(30*time.Minute), unitIDs)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != constants.CodeStateConflict {
		t.Fatalf("overlapping window must conflict, got %v", err)
	}
	if got := appErr.Message; !contains(got, "CA100") || !contains(got, "TUG-01") {
		t.Fatalf("conflict message must name flight and unit, got %q", got)
	}
	// 3. Back-to-back at the window edge is allowed.
	if err := createTurnaround("CA300", base.Add(90*time.Minute), unitIDs); err != nil {
		t.Fatalf("back-to-back turnaround should be allowed: %v", err)
	}
	// 4. A different unit stays free for the same slot.
	if err := createTurnaround("CA400", base.Add(30*time.Minute), []string{itoa(units[1].ID)}); err != nil {
		t.Fatalf("other unit should be schedulable: %v", err)
	}
	// 5. Completing the first turnaround releases its window.
	var first model.Turnaround
	if err := db.Where("flight_no = ?", "CA100").First(&first).Error; err != nil {
		t.Fatalf("load first turnaround: %v", err)
	}
	if err := db.Model(&model.Turnaround{}).Where("id = ?", first.ID).Update("status", constants.TurnaroundCompleted).Error; err != nil {
		t.Fatalf("complete first turnaround: %v", err)
	}
	if err := createTurnaround("CA500", base, unitIDs); err != nil {
		t.Fatalf("completed turnaround must release the window: %v", err)
	}
	// 6. A revoked clearance also releases the window.
	var third model.Turnaround
	if err := db.Where("flight_no = ?", "CA300").First(&third).Error; err != nil {
		t.Fatalf("load third turnaround: %v", err)
	}
	if err := db.Model(&model.Turnaround{}).Where("id = ?", third.ID).Update("status", constants.TurnaroundDecisioned).Error; err != nil {
		t.Fatalf("mark decisioned: %v", err)
	}
	if err := db.Model(&model.ClearanceDecision{}).Where("turnaround_id = ?", third.ID).Update("state", constants.ClearanceRevoked).Error; err != nil {
		t.Fatalf("revoke clearance: %v", err)
	}
	if err := createTurnaround("CA600", base.Add(100*time.Minute), unitIDs); err != nil {
		t.Fatalf("revoked turnaround must release the window: %v", err)
	}
	// 7. Occupancy board reports the next window per unit and free units.
	rows, total, err := turnaroundSvc.List(1, 10, "", "", "")
	if err != nil || total == 0 {
		t.Fatalf("list turnarounds: %v total=%d", err, total)
	}
	states := make(map[string]string, len(rows))
	for _, row := range rows {
		states[row.FlightNo] = row.ClearanceState
	}
	if states["CA100"] != constants.ClearancePending || states["CA300"] != constants.ClearanceRevoked {
		t.Fatalf("list must expose clearance states, got %#v", states)
	}
	occupancy, err := unitSvc.Occupancy()
	if err != nil {
		t.Fatalf("occupancy: %v", err)
	}
	items, ok := occupancy["items"].([]model.UnitOccupancy)
	if !ok || len(items) != 2 {
		t.Fatalf("unexpected occupancy payload: %#v", occupancy)
	}
	byUnit := make(map[uint64]model.UnitOccupancy, len(items))
	for _, item := range items {
		byUnit[item.UnitID] = item
	}
	// CA500 (base .. base+90min) is the earliest window still open for TUG-01.
	tug := byUnit[units[0].ID]
	if tug.WindowStart == nil || tug.FlightNo != "CA500" {
		t.Fatalf("expected next TUG-01 window from CA500, got %#v", tug)
	}
	if want := base.Add(90 * time.Minute); !tug.WindowEnd.Equal(want) {
		t.Fatalf("window end = %v, want %v", *tug.WindowEnd, want)
	}
	gpu := byUnit[units[1].ID]
	if gpu.WindowStart == nil || gpu.FlightNo != "CA400" {
		t.Fatalf("expected next GPU-01 window from CA400, got %#v", gpu)
	}
	// 8. Once every window expired or released, the unit shows up free.
	if err := db.Model(&model.Turnaround{}).Where("flight_no IN ?", []string{"CA400", "CA500", "CA600"}).
		Update("status", constants.TurnaroundCompleted).Error; err != nil {
		t.Fatalf("complete remaining turnarounds: %v", err)
	}
	occupancy, err = unitSvc.Occupancy()
	if err != nil {
		t.Fatalf("occupancy after completion: %v", err)
	}
	items = occupancy["items"].([]model.UnitOccupancy)
	for _, item := range items {
		if item.WindowStart != nil {
			t.Fatalf("unit %s should be free, got window %#v", item.UnitCode, item)
		}
	}
}

func itoa(id uint64) string {
	return strconv.FormatUint(id, 10)
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
