package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/model"
)

type OrganizationResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	OwnerID   uuid.UUID `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	MyRole    string    `json:"my_role,omitempty"`
}

func NewOrganizationResponseDTO(org *model.Organization) *OrganizationResponseDTO {
	return &OrganizationResponseDTO{
		ID:        org.ID,
		Name:      org.Name,
		Status:    string(org.Status),
		OwnerID:   org.OwnerID,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}
}

func NeworganizationListDTO(list []model.OrganizationWithRole) []OrganizationResponseDTO {
	out := make([]OrganizationResponseDTO, len(list))
	for i, item := range list {
		out[i] = OrganizationResponseDTO{
			ID:        item.ID,
			Name:      item.Name,
			Status:    string(item.Status),
			OwnerID:   item.OwnerID,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			MyRole:    item.MyRole,
		}
	}
	return out
}

type MyRoleResponseDTO struct {
	Role string `json:"role"`
}
