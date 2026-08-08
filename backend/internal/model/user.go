package model

import (
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserPending  UserStatus = "pending"
	UserActive   UserStatus = "active"
	UserDisabled UserStatus = "disabled"
)

type User struct {
	ID        uuid.UUID
	Name      string
	Email     string
	Status    UserStatus
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
