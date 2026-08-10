package dto

import (
	"time"

	"github.com/nnieru/mini-evvy/internal/model"
)

type BookingListItemDTO struct {
	ID            string     `json:"id"`
	GuestID       string     `json:"guest_id"`
	EventID       string     `json:"event_id"`
	CategoryID    string     `json:"category_id"`
	SeatID        string     `json:"seat_id"`
	Source        string     `json:"source"`
	Status        string     `json:"status"`
	Notes         *string    `json:"notes,omitempty"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	Barcode                     *string    `json:"barcode,omitempty"`
	InvitationEmailStatus       string     `json:"invitation_email_status"`
	InvitationEmailSentAt       *time.Time `json:"invitation_email_sent_at,omitempty"`
	InvitationResendAvailableAt *time.Time `json:"invitation_resend_available_at,omitempty"`
	CreatedBy                   string     `json:"created_by"`
	UpdatedBy     *string    `json:"updated_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	GuestName     string     `json:"guest_name"`
	GuestEmail    string     `json:"guest_email"`
	SeatCode      string     `json:"seat_code"`
	PaymentLabel  string     `json:"payment_label"`
}

type PaginatedBookingListDTO struct {
	Items    []BookingListItemDTO `json:"items"`
	Total    int                  `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

func paymentLabel(status model.BookingStatus) string {
	if status == model.BookingPaid {
		return "paid"
	}
	return "unpaid"
}

func NewBookingListItemDTO(row *model.BookingListRow) BookingListItemDTO {
	base := NewBookingResponseDTO(&row.SeatBooking)
	dto := BookingListItemDTO{
		ID:           base.ID,
		GuestID:      base.GuestID,
		EventID:      base.EventID,
		CategoryID:   base.CategoryID,
		SeatID:       base.SeatID,
		Source:       base.Source,
		Status:       base.Status,
		Notes:        base.Notes,
		PaidAt:       base.PaidAt,
		Barcode:                     base.Barcode,
		InvitationEmailStatus:       base.InvitationEmailStatus,
		InvitationEmailSentAt:       base.InvitationEmailSentAt,
		InvitationResendAvailableAt: base.InvitationResendAvailableAt,
		CreatedBy:                   base.CreatedBy,
		UpdatedBy:    base.UpdatedBy,
		CreatedAt:    base.CreatedAt,
		UpdatedAt:    base.UpdatedAt,
		GuestName:    row.GuestName,
		GuestEmail:   row.GuestEmail,
		SeatCode:     row.SeatCode,
		PaymentLabel: paymentLabel(row.Status),
	}
	return dto
}

func NewPaginatedBookingListDTO(rows []model.BookingListRow, total, page, pageSize int) PaginatedBookingListDTO {
	items := make([]BookingListItemDTO, 0, len(rows))
	for i := range rows {
		items = append(items, NewBookingListItemDTO(&rows[i]))
	}
	return PaginatedBookingListDTO{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}
