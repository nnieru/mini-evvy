package model

import "github.com/google/uuid"

type SeatingPreviewRow struct {
	DraftItemID  uuid.UUID
	GuestName    string
	GuestEmail   string
	SeatCode     string
	SeatID       uuid.UUID
	CategoryName string
	CategoryCode *string
	Section      *string
}
