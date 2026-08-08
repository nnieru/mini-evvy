package dto

import (
	"time"

	"github.com/nnieru/mini-evvy/internal/model"
)

type BookingResponseDTO struct {
	ID         string     `json:"id"`
	GuestID    string     `json:"guest_id"`
	EventID    string     `json:"event_id"`
	CategoryID string     `json:"category_id"`
	SeatID     string     `json:"seat_id"`
	Source     string     `json:"source"`
	Status     string     `json:"status"`
	Notes      *string    `json:"notes,omitempty"`
	PaidAt     *time.Time `json:"paid_at,omitempty"`
	Barcode    *string    `json:"barcode,omitempty"`
	CreatedBy  string     `json:"created_by"`
	UpdatedBy  *string    `json:"updated_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func NewBookingResponseDTO(b *model.SeatBooking) BookingResponseDTO {
	dto := BookingResponseDTO{
		ID:         b.ID.String(),
		GuestID:    b.GuestID.String(),
		EventID:    b.EventID.String(),
		CategoryID: b.CategoryID.String(),
		SeatID:     b.SeatID.String(),
		Source:     string(b.Source),
		Status:     string(b.Status),
		Notes:      b.Notes,
		PaidAt:     b.PaidAt,
		Barcode:    b.Barcode,
		CreatedBy:  b.CreatedBy.String(),
		CreatedAt:  b.CreatedAt,
		UpdatedAt:  b.UpdatedAt,
	}
	if b.UpdatedBy != nil {
		s := b.UpdatedBy.String()
		dto.UpdatedBy = &s
	}
	return dto
}

func NewBookingListDTO(list []model.SeatBooking) []BookingResponseDTO {
	out := make([]BookingResponseDTO, 0, len(list))
	for i := range list {
		out = append(out, NewBookingResponseDTO(&list[i]))
	}
	return out
}
