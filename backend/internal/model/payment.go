package model

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentPending  PaymentStatus = "pending"
	PaymentSuccess  PaymentStatus = "success"
	PaymentFailed   PaymentStatus = "failed"
	PaymentRefunded PaymentStatus = "refunded"
)

type Payment struct {
	ID         uuid.UUID
	BookingID  uuid.UUID
	Amount     string
	Currency   string
	Method     *string
	GatewayRef *string
	Status     PaymentStatus
	PaidAt     *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
