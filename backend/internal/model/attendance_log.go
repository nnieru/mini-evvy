package model

import (
	"time"

	"github.com/google/uuid"
)

type AttendanceStatus string

const (
	AttendanceStatusCheckedIn    AttendanceStatus = "checked_in"
	AttendanceStatusNotCheckedIn AttendanceStatus = "not_checked_in"
)

type AttendanceLog struct {
	ID        uuid.UUID
	GuestID   uuid.UUID
	EventID   uuid.UUID
	SeatID    uuid.UUID
	Status    AttendanceStatus
	Message   *string
	CreatedBy uuid.UUID
	UpdatedBy *uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
