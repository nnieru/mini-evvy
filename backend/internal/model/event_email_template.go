package model

import (
	"time"

	"github.com/google/uuid"
)

const EmailTemplateTypeInvitation = "invitation"

type EventEmailTemplate struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	Type      string
	Config    []byte
	CreatedAt time.Time
	UpdatedAt time.Time
	UpdatedBy *uuid.UUID
}
