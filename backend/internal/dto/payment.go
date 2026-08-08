package dto

import (
	"time"

	"github.com/nnieru/mini-evvy/internal/model"
)

type PaymentResponseDTO struct {
	ID         string     `json:"id"`
	BookingID  string     `json:"booking_id"`
	Amount     string     `json:"amount"`
	Currency   string     `json:"currency"`
	Method     *string    `json:"method,omitempty"`
	GatewayRef *string    `json:"gateway_ref,omitempty"`
	Status     string     `json:"status"`
	PaidAt     *time.Time `json:"paid_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func NewPaymentResponseDTO(p *model.Payment) PaymentResponseDTO {
	return PaymentResponseDTO{
		ID:         p.ID.String(),
		BookingID:  p.BookingID.String(),
		Amount:     p.Amount,
		Currency:   p.Currency,
		Method:     p.Method,
		GatewayRef: p.GatewayRef,
		Status:     string(p.Status),
		PaidAt:     p.PaidAt,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

func NewPaymentListDTO(list []model.Payment) []PaymentResponseDTO {
	out := make([]PaymentResponseDTO, 0, len(list))
	for i := range list {
		out = append(out, NewPaymentResponseDTO(&list[i]))
	}
	return out
}
