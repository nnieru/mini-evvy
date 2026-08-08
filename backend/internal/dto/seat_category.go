package dto

import (
	"time"

	"github.com/nnieru/mini-evvy/internal/model"
)

type SeatCategoryResponseDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      *string   `json:"code,omitempty"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency"`
	EventID   string    `json:"event_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewSeatCategoryResponseDTO(c *model.SeatCategory) SeatCategoryResponseDTO {
	return SeatCategoryResponseDTO{
		ID:        c.ID.String(),
		Name:      c.Name,
		Code:      c.Code,
		Price:     c.Price,
		Currency:  c.Currency,
		EventID:   c.EventID.String(),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func NewSeatCategoryListDTO(list []model.SeatCategory) []SeatCategoryResponseDTO {
	out := make([]SeatCategoryResponseDTO, 0, len(list))
	for i := range list {
		out = append(out, NewSeatCategoryResponseDTO(&list[i]))
	}
	return out
}
