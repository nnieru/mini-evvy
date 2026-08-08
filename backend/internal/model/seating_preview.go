package model

import "github.com/google/uuid"

type SeatingPreviewRow struct {
	BookingID    uuid.UUID
	GuestName    string
	GuestEmail   string
	SeatCode     string
	CategoryName string
	CategoryCode *string
	Section      *string
}
