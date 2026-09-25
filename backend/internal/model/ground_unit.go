package model

import "time"

// GroundUnit is an airside vehicle or powered support unit.
type GroundUnit struct {
	ID               uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UnitCode         string     `gorm:"size:40;uniqueIndex;not null" json:"unit_code"`
	Name             string     `gorm:"size:120;not null" json:"name"`
	UnitType         string     `gorm:"size:40;not null;index" json:"unit_type"`
	Stand            string     `gorm:"size:30;not null;index" json:"stand"`
	State            string     `gorm:"size:30;not null;default:available;index;check:chk_ground_units_state,state IN ('available','inspection','blocked','retired')" json:"state"`
	LastInspectionAt *time.Time `json:"last_inspection_at"`
	Notes            string     `gorm:"type:text" json:"notes"`
	Version          int        `gorm:"not null;default:1" json:"version"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (GroundUnit) TableName() string { return "ground_units" }

// UnitOccupancy is the next reservation window of one ground unit. The
// equipment board uses it to show what occupies the unit next, or that the
// unit is free when no window is present.
type UnitOccupancy struct {
	UnitID       uint64     `json:"unit_id"`
	UnitCode     string     `json:"unit_code"`
	State        string     `json:"state"`
	OccupiedNow  bool       `json:"occupied_now"`
	TurnaroundID uint64     `json:"turnaround_id,omitempty"`
	FlightNo     string     `json:"flight_no,omitempty"`
	WindowStart  *time.Time `json:"window_start"`
	WindowEnd    *time.Time `json:"window_end"`
}
