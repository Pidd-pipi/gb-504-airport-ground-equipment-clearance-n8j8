package dto

import "time"

type TurnaroundCreateRequest struct {
	FlightNo      string               `json:"flight_no" binding:"required,max=20"`
	Stand         string               `json:"stand" binding:"required,max=30"`
	Phase         string               `json:"phase" binding:"required,oneof=arrival servicing departure"`
	ScheduledAt   time.Time            `json:"scheduled_at" binding:"required"`
	RiskLevel     string               `json:"risk_level" binding:"required,oneof=low medium high critical"`
	GroundUnitIDs []string             `json:"ground_unit_ids"`
	CoordinatorID uint64               `json:"coordinator_id" binding:"required"`
	Checks        []SafetyCheckRequest `json:"checks" binding:"required,min=1"`
}

type TurnaroundStatusRequest struct {
	Status  string `json:"status" binding:"required,oneof=open checking decisioned completed"`
	Version int    `json:"version" binding:"required,min=1"`
}

type SafetyCheckRequest struct {
	GroundUnitID *uint64  `json:"ground_unit_id"`
	CheckCode    string   `json:"check_code" binding:"required,max=40"`
	ItemName     string   `json:"item_name" binding:"required,max=200"`
	RiskLevel    string   `json:"risk_level" binding:"required,oneof=low medium high critical"`
	Evidence     []string `json:"evidence"`
}

type SafetyCheckReviewRequest struct {
	Result   string   `json:"result" binding:"required,oneof=passed failed"`
	Evidence []string `json:"evidence" binding:"required,min=1"`
	Remark   string   `json:"remark" binding:"max=1000"`
}

// OccupancyWindow is one planned 90-minute occupation segment on a ground unit.
type OccupancyWindow struct {
	TurnaroundID uint64    `json:"turnaround_id"`
	FlightNo     string    `json:"flight_no"`
	Stand        string    `json:"stand"`
	Status       string    `json:"status"`
	StartAt      time.Time `json:"start_at"`
	EndAt        time.Time `json:"end_at"`
}

// UnitOccupancy is the scheduling read model for one ground unit. Completed
// turnarounds and revoked clearance decisions never occupy the schedule.
type UnitOccupancy struct {
	UnitID         uint64            `json:"unit_id"`
	UnitCode       string            `json:"unit_code"`
	ReserveMinutes int               `json:"reserve_minutes"`
	Current        *OccupancyWindow  `json:"current"`
	Next           *OccupancyWindow  `json:"next"`
	FreeAt         *time.Time        `json:"free_at"`
	Windows        []OccupancyWindow `json:"windows"`
}

// OccupancyBoard groups the per-unit schedule by unit id.
type OccupancyBoard struct {
	ReserveMinutes int                       `json:"reserve_minutes"`
	Units          map[uint64]*UnitOccupancy `json:"units"`
}
