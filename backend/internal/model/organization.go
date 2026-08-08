package model

import (
	"time"

	"github.com/google/uuid"
)

type OrganizationStatus string

const (
	OrganizationActive   OrganizationStatus = "active"
	OrganizationInReview OrganizationStatus = "in_review"
	OrganizationDisabled OrganizationStatus = "disabled"
)

type Organization struct {
	ID        uuid.UUID
	Name      string
	Status    OrganizationStatus
	OwnerID   uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
