package dto

import (
	"time"

	"github.com/nnieru/mini-evvy/internal/model"
)

type EventResponseDTO struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	SeatingPhase   string  `json:"seating_phase"`
	Description    *string `json:"description,omitempty"`
	StartDate      string  `json:"start_date"`
	EndDate        *string `json:"end_date,omitempty"`
	StartTime      *string `json:"start_time,omitempty"`
	EndTime        *string `json:"end_time,omitempty"`
	CreatorID      string  `json:"creator_id"`
	OrganizationID string  `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func NewEventResponseDTO(e *model.Event) EventResponseDTO {
	dto := EventResponseDTO{
		ID:             e.ID.String(),
		Name:           e.Name,
		Status:         string(e.Status),
		SeatingPhase:   string(e.SeatingPhase),
		Description:    e.Description,
		StartDate:      e.StartDate.Format("2006-01-02"),
		StartTime:      e.StartTime,
		EndTime:        e.EndTime,
		CreatorID:      e.CreatorID.String(),
		OrganizationID: e.OrganizationID.String(),
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
	if e.EndDate != nil {
		s := e.EndDate.Format("2006-01-02")
		dto.EndDate = &s
	}
	return dto
}

func NewEventListDTO(list []model.Event) []EventResponseDTO {
	out := make([]EventResponseDTO, 0, len(list))
	for i := range list {
		out = append(out, NewEventResponseDTO(&list[i]))
	}
	return out
}

type MyEventResponseDTO struct {
	EventResponseDTO
	OrganizationName string `json:"organization_name"`
}

func NewMyEventResponseDTO(e *model.EventWithOrganization) MyEventResponseDTO {
	return MyEventResponseDTO{
		EventResponseDTO: NewEventResponseDTO(&e.Event),
		OrganizationName: e.OrganizationName,
	}
}

func NewMyEventListDTO(list []model.EventWithOrganization) []MyEventResponseDTO {
	out := make([]MyEventResponseDTO, 0, len(list))
	for i := range list {
		out = append(out, NewMyEventResponseDTO(&list[i]))
	}
	return out
}
