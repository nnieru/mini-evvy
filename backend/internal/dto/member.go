package dto

import (
	"time"

	"github.com/nnieru/mini-evvy/internal/model"
)

type MemberResponseDTO struct {
	UserRoleID     string    `json:"user_role_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	RoleID         string    `json:"role_id"`
	RoleName       string    `json:"role_name"`
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewMemberResponseDTO(m *model.Member) MemberResponseDTO {
	return MemberResponseDTO{
		UserRoleID:     m.UserRoleID.String(),
		Name:           m.Name,
		Email:          m.Email,
		RoleID:         m.RoleID.String(),
		RoleName:       m.RoleName,
		OrganizationID: m.OrganizationID.String(),
		CreatedAt:      m.CreatedAt,
	}
}

func NewMemberListDTO(list []model.Member) []MemberResponseDTO {
	out := make([]MemberResponseDTO, 0, len(list))

	for i := range list {
		out = append(out, NewMemberResponseDTO(&list[i]))
	}

	return out
}
