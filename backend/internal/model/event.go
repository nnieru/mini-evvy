package model

import (
	"time"

	"github.com/google/uuid"
)

type EventStatus string

const (
	EventActive    EventStatus = "active"
	EventDraft     EventStatus = "draft"
	EventCancelled EventStatus = "cancelled"
	EventCompleted EventStatus = "completed"
)

type SeatingPhase string

const (
	SeatingOpen     SeatingPhase = "open"
	SeatingPreview  SeatingPhase = "preview"
	SeatingApproved SeatingPhase = "approved"
)

type Event struct {
	ID             uuid.UUID
	Name           string
	Status         EventStatus
	SeatingPhase   SeatingPhase
	Description    *string
	StartDate      time.Time
	EndDate        *time.Time
	StartTime      *string
	EndTime        *string
	CreatorID      uuid.UUID
	OrganizationID uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}
