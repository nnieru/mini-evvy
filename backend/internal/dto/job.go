package dto

import (
	"encoding/json"
	"time"

	"github.com/nnieru/mini-evvy/internal/model"
)

type JobResponseDTO struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Status     string          `json:"status"`
	RetryCount int             `json:"retry_count"`
	Data       json.RawMessage `json:"data,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func NewJobResponseDTO(j *model.Job) JobResponseDTO {
	return JobResponseDTO{
		ID:         j.ID.String(),
		Type:       j.Type,
		Status:     string(j.Status),
		RetryCount: j.RetryCount,
		Data:       j.Data,
		CreatedAt:  j.CreatedAt,
		UpdatedAt:  j.UpdatedAt,
	}
}

func NewJobListDTO(list []model.Job) []JobResponseDTO {
	out := make([]JobResponseDTO, 0, len(list))
	for i := range list {
		out = append(out, NewJobResponseDTO(&list[i]))
	}
	return out
}
