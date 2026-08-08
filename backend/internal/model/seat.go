package model

import (
	"time"

	"github.com/google/uuid"
)

type SeatStatus string

const (
	SeatAvailable SeatStatus = "available"
	SeatReserved  SeatStatus = "reserved"
	SeatOccupied  SeatStatus = "occupied"
	SeatBlocked   SeatStatus = "blocked"
)

type Seat struct {
	ID          uuid.UUID
	Code        string
	Description *string
	Section     *string
	Row         *int
	Col         *int
	Status      SeatStatus
	EventID     uuid.UUID
	CategoryID  uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
