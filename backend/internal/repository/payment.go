package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
)

type PaymentRepo struct {
	pool *pgxpool.Pool
}

func NewPaymentRepo(pool *pgxpool.Pool) *PaymentRepo {
	return &PaymentRepo{pool: pool}
}

func (r *PaymentRepo) Create(ctx context.Context, db DBTX, p *model.Payment) (*model.Payment, error) {
	const query = `
		INSERT INTO payments (
			booking_id, amount, currency, method, gateway_ref, status, paid_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, booking_id, amount::text, currency, method, gateway_ref, status,
			paid_at, created_at, updated_at
	`

	var payment model.Payment
	err := db.QueryRow(ctx, query,
		p.BookingID,
		p.Amount,
		p.Currency,
		p.Method,
		p.GatewayRef,
		p.Status,
		p.PaidAt,
	).Scan(
		&payment.ID,
		&payment.BookingID,
		&payment.Amount,
		&payment.Currency,
		&payment.Method,
		&payment.GatewayRef,
		&payment.Status,
		&payment.PaidAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	return &payment, nil
}

func (r *PaymentRepo) ListByBookingID(ctx context.Context, db DBTX, bookingID uuid.UUID) ([]model.Payment, error) {
	const query = `
		SELECT id, booking_id, amount::text, currency, method, gateway_ref, status,
			paid_at, created_at, updated_at
		FROM payments
		WHERE booking_id = $1
		ORDER BY created_at ASC
	`

	rows, err := db.Query(ctx, query, bookingID)
	if err != nil {
		return nil, fmt.Errorf("list payments by booking id: %w", err)
	}
	defer rows.Close()

	var out []model.Payment
	for rows.Next() {
		var payment model.Payment
		if err := rows.Scan(
			&payment.ID,
			&payment.BookingID,
			&payment.Amount,
			&payment.Currency,
			&payment.Method,
			&payment.GatewayRef,
			&payment.Status,
			&payment.PaidAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		out = append(out, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list payments rows: %w", err)
	}

	return out, nil
}
