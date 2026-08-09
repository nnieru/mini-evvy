package dto

import (
	"time"

	"github.com/nnieru/mini-evvy/internal/model"
)

type AttendanceResponseDTO struct {
	ID        string     `json:"id"`
	GuestID   string     `json:"guest_id"`
	EventID   string     `json:"event_id"`
	SeatID    string     `json:"seat_id"`
	Status    string     `json:"status"`
	Message   *string    `json:"message,omitempty"`
	CreatedBy string     `json:"created_by"`
	UpdatedBy *string    `json:"updated_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func NewAttendanceResponseDTO(a *model.AttendanceLog) AttendanceResponseDTO {
	dto := AttendanceResponseDTO{
		ID:        a.ID.String(),
		GuestID:   a.GuestID.String(),
		EventID:   a.EventID.String(),
		SeatID:    a.SeatID.String(),
		Status:    string(a.Status),
		Message:   a.Message,
		CreatedBy: a.CreatedBy.String(),
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
	if a.UpdatedBy != nil {
		s := a.UpdatedBy.String()
		dto.UpdatedBy = &s
	}
	return dto
}

func NewAttendanceListDTO(list []model.AttendanceLog) []AttendanceResponseDTO {
	out := make([]AttendanceResponseDTO, 0, len(list))
	for i := range list {
		out = append(out, NewAttendanceResponseDTO(&list[i]))
	}
	return out
}

func NewPaginatedAttendanceListDTO(list []model.AttendanceLog, total, page, pageSize int) PaginatedList[AttendanceResponseDTO] {
	return NewPaginatedList(NewAttendanceListDTO(list), total, page, pageSize)
}
