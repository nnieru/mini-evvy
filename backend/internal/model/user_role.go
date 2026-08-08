package model

import (
	"time"

	"github.com/google/uuid"
)

type UserRole struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	RoleID         uuid.UUID
	OrganizationID uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Member is a joined view of user_roles + users + roles.
type Member struct {
	UserRoleID     uuid.UUID
	UserID         uuid.UUID
	Name           string
	Email          string
	RoleID         uuid.UUID
	RoleName       string
	OrganizationID uuid.UUID
	CreatedAt      time.Time
}
