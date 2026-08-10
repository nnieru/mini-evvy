package model

import (
	"time"

	"github.com/google/uuid"
)

type SeatingDraftStatus string

const (
	SeatingDraftOpen     SeatingDraftStatus = "open"
	SeatingDraftApproved SeatingDraftStatus = "approved"
	SeatingDraftRejected SeatingDraftStatus = "rejected"
)

type SeatingDraft struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	Status    SeatingDraftStatus
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SeatingDraftItem struct {
	ID         uuid.UUID
	DraftID    uuid.UUID
	GuestID    uuid.UUID
	SeatID     uuid.UUID
	CategoryID uuid.UUID
	CreatedAt  time.Time
}
