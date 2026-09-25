package model

import (
	"time"

	"groundclearance/internal/constants"
)

// Turnaround tracks one operational phase for an arriving or departing flight.
type Turnaround struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	FlightNo      string    `gorm:"size:20;not null;index" json:"flight_no"`
	Stand         string    `gorm:"size:30;not null;index" json:"stand"`
	Phase         string    `gorm:"size:30;not null" json:"phase"`
	ScheduledAt   time.Time `gorm:"index" json:"scheduled_at"`
	RiskLevel     string    `gorm:"size:20;not null;default:medium;index;check:chk_turnarounds_risk,risk_level IN ('low','medium','high','critical')" json:"risk_level"`
	Status        string    `gorm:"size:30;not null;default:open;index;check:chk_turnarounds_status,status IN ('open','checking','decisioned','completed')" json:"status"`
	GroundUnitIDs JSONList  `gorm:"type:jsonb;index:idx_turnarounds_ground_unit_ids,type:gin" json:"ground_unit_ids"`
	CoordinatorID uint64    `gorm:"not null;index" json:"coordinator_id"`
	Version       int       `gorm:"not null;default:1" json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Turnaround) TableName() string { return "turnarounds" }

// WindowStart is the beginning of the planned occupation window for assigned units.
func (t Turnaround) WindowStart() time.Time { return t.ScheduledAt }

// WindowEnd is the exclusive end of the planned occupation window. Two windows
// are allowed to chain back-to-back when one ends exactly when the next starts.
func (t Turnaround) WindowEnd() time.Time {
	return t.ScheduledAt.Add(constants.ScheduleWindowMinutes * time.Minute)
}
