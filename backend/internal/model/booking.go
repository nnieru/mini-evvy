package model

import (
	"time"

	"github.com/google/uuid"
)

type BookingSource string

const (
	SourceInvited BookingSource = "invited"
	SourceOnsite  BookingSource = "onsite"
)

type BookingStatus string

const (
	BookingPending   BookingStatus = "pending"
	BookingNotPaid   BookingStatus = "not_paid"
	BookingPaid      BookingStatus = "paid"
	BookingCancelled BookingStatus = "cancelled"
)

type SeatBooking struct {
	ID         uuid.UUID
	GuestID    uuid.UUID
	EventID    uuid.UUID
	CategoryID uuid.UUID
	SeatID     uuid.UUID
	Source     BookingSource
	Status     BookingStatus
	Notes      *string
	PaidAt     *time.Time
	Barcode    *string
	CreatedBy  uuid.UUID
	UpdatedBy  *uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}
