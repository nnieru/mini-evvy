package dto

import "github.com/nnieru/mini-evvy/internal/model"

type SeatingPreviewRowDTO struct {
	BookingID    string  `json:"booking_id"`
	GuestName    string  `json:"guest_name"`
	GuestEmail   string  `json:"guest_email"`
	SeatCode     string  `json:"seat_code"`
	CategoryName string  `json:"category_name"`
	CategoryCode *string `json:"category_code,omitempty"`
	Section      *string `json:"section,omitempty"`
}

func NewSeatingPreviewRowDTO(row model.SeatingPreviewRow) SeatingPreviewRowDTO {
	return SeatingPreviewRowDTO{
		BookingID:    row.BookingID.String(),
		GuestName:    row.GuestName,
		GuestEmail:   row.GuestEmail,
		SeatCode:     row.SeatCode,
		CategoryName: row.CategoryName,
		CategoryCode: row.CategoryCode,
		Section:      row.Section,
	}
}

func NewSeatingPreviewListDTO(rows []model.SeatingPreviewRow) []SeatingPreviewRowDTO {
	out := make([]SeatingPreviewRowDTO, 0, len(rows))
	for i := range rows {
		out = append(out, NewSeatingPreviewRowDTO(rows[i]))
	}
	return out
}
