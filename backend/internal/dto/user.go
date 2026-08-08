package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/nnieru/mini-evvy/internal/model"
)

type UserResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

func NewUserResponseDTO(user *model.User) UserResponseDTO {
	return UserResponseDTO{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		Status:    string(user.Status),
	}
}
