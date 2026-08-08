package dto

import (
	"time"

	"github.com/nnieru/mini-evvy/internal/model"
)

type GuestResponseDTO struct {
	ID          string     `json:"id"`
	EventID     string     `json:"event_id"`
	CategoryID  string     `json:"category_id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	PaidDate    *time.Time `json:"paid_date,omitempty"`
	TicketCount int        `json:"ticket_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func NewGuestResponseDTO(g *model.Guest) GuestResponseDTO {
	return GuestResponseDTO{
		ID:          g.ID.String(),
		EventID:     g.EventID.String(),
		CategoryID:  g.CategoryID.String(),
		Name:        g.Name,
		Email:       g.Email,
		PaidDate:    g.PaidDate,
		TicketCount: g.TicketCount,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

func NewGuestListDTO(list []model.Guest) []GuestResponseDTO {
	out := make([]GuestResponseDTO, 0, len(list))
	for i := range list {
		out = append(out, NewGuestResponseDTO(&list[i]))
	}
	return out
}
