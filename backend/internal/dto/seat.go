package dto

import (
	"time"

	"github.com/nnieru/mini-evvy/internal/model"
)

type SeatResponseDTO struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Section     *string `json:"section,omitempty"`
	Row         *int    `json:"row,omitempty"`
	Col         *int    `json:"col,omitempty"`
	Status      string  `json:"status"`
	Description *string `json:"description,omitempty"`
	EventID     string  `json:"event_id"`
	CategoryID  string  `json:"category_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewSeatResponseDTO(s *model.Seat) SeatResponseDTO {
	return SeatResponseDTO{
		ID:          s.ID.String(),
		Code:        s.Code,
		Section:     s.Section,
		Row:         s.Row,
		Col:         s.Col,
		Status:      string(s.Status),
		Description: s.Description,
		EventID:     s.EventID.String(),
		CategoryID:  s.CategoryID.String(),
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func NewSeatListDTO(list []model.Seat) []SeatResponseDTO {
	out := make([]SeatResponseDTO, 0, len(list))
	for i := range list {
		out = append(out, NewSeatResponseDTO(&list[i]))
	}
	return out
}

func NewPaginatedSeatListDTO(list []model.Seat, total, page, pageSize int) PaginatedList[SeatResponseDTO] {
	return NewPaginatedList(NewSeatListDTO(list), total, page, pageSize)
}
