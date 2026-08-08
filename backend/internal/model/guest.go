package model

import (
	"time"

	"github.com/google/uuid"
)

type Guest struct {
	ID          uuid.UUID
	EventID     uuid.UUID
	CategoryID  uuid.UUID
	Name        string
	Email       string
	PaidDate    *time.Time
	TicketCount int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
