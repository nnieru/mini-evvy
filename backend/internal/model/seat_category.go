package model

import (
	"time"

	"github.com/google/uuid"
)

type SeatCategory struct {
	ID        uuid.UUID
	Name      string
	Code      *string
	Price     float64
	Currency  string
	EventID   uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
