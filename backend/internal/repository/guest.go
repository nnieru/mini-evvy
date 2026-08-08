package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
)

type GuestRepo struct {
	pool *pgxpool.Pool
}

func NewGuestRepo(pool *pgxpool.Pool) *GuestRepo {
	return &GuestRepo{pool: pool}
}

func (r *GuestRepo) Create(ctx context.Context, db DBTX, g *model.Guest) (*model.Guest, error) {
	const query = `
		INSERT INTO guests (event_id, category_id, name, email, paid_date, ticket_count)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, event_id, category_id, name, email, paid_date, ticket_count,
			created_at, updated_at, deleted_at
	`

	var guest model.Guest
	err := db.QueryRow(ctx, query,
		g.EventID,
		g.CategoryID,
		g.Name,
		g.Email,
		g.PaidDate,
		g.TicketCount,
	).Scan(
		&guest.ID,
		&guest.EventID,
		&guest.CategoryID,
		&guest.Name,
		&guest.Email,
		&guest.PaidDate,
		&guest.TicketCount,
		&guest.CreatedAt,
		&guest.UpdatedAt,
		&guest.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create guest: %w", err)
	}

	return &guest, nil
}

func (r *GuestRepo) GetByID(ctx context.Context, db DBTX, id uuid.UUID) (*model.Guest, error) {
	const query = `
		SELECT id, event_id, category_id, name, email, paid_date, ticket_count,
			created_at, updated_at, deleted_at
		FROM guests
		WHERE id = $1 AND deleted_at IS NULL
	`

	var guest model.Guest
	err := db.QueryRow(ctx, query, id).Scan(
		&guest.ID,
		&guest.EventID,
		&guest.CategoryID,
		&guest.Name,
		&guest.Email,
		&guest.PaidDate,
		&guest.TicketCount,
		&guest.CreatedAt,
		&guest.UpdatedAt,
		&guest.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get guest by id: %w", err)
	}

	return &guest, nil
}

func (r *GuestRepo) GetByEventNameEmailCategory(
	ctx context.Context,
	db DBTX,
	eventID, categoryID uuid.UUID,
	name, email string,
) (*model.Guest, error) {
	const query = `
		SELECT id, event_id, category_id, name, email, paid_date, ticket_count,
			created_at, updated_at, deleted_at
		FROM guests
		WHERE event_id = $1
			AND category_id = $2
			AND lower(trim(email)) = lower(trim($3))
			AND lower(trim(name)) = lower(trim($4))
			AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT 1
	`

	var guest model.Guest
	err := db.QueryRow(ctx, query, eventID, categoryID, email, name).Scan(
		&guest.ID,
		&guest.EventID,
		&guest.CategoryID,
		&guest.Name,
		&guest.Email,
		&guest.PaidDate,
		&guest.TicketCount,
		&guest.CreatedAt,
		&guest.UpdatedAt,
		&guest.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get guest by event name email category: %w", err)
	}

	return &guest, nil
}

func (r *GuestRepo) ListByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) ([]model.Guest, error) {
	const query = `
		SELECT id, event_id, category_id, name, email, paid_date, ticket_count,
			created_at, updated_at, deleted_at
		FROM guests
		WHERE event_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := db.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("list guests by event id: %w", err)
	}
	defer rows.Close()

	var out []model.Guest
	for rows.Next() {
		var guest model.Guest
		if err := rows.Scan(
			&guest.ID,
			&guest.EventID,
			&guest.CategoryID,
			&guest.Name,
			&guest.Email,
			&guest.PaidDate,
			&guest.TicketCount,
			&guest.CreatedAt,
			&guest.UpdatedAt,
			&guest.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan guest: %w", err)
		}
		out = append(out, guest)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list guests rows: %w", err)
	}

	return out, nil
}

func (r *GuestRepo) ListUnbookedByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) ([]model.Guest, error) {
	const query = `
		SELECT g.id, g.event_id, g.category_id, g.name, g.email, g.paid_date, g.ticket_count,
			g.created_at, g.updated_at, g.deleted_at
		FROM guests g
		WHERE g.event_id = $1 AND g.deleted_at IS NULL
			AND (
				SELECT COUNT(*) FROM seat_bookings sb
				WHERE sb.guest_id = g.id AND sb.deleted_at IS NULL AND sb.status <> 'cancelled'
			) < g.ticket_count
		ORDER BY g.paid_date ASC NULLS LAST, g.created_at ASC
	`

	rows, err := db.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("list unbooked guests by event id: %w", err)
	}
	defer rows.Close()

	var out []model.Guest
	for rows.Next() {
		var guest model.Guest
		if err := rows.Scan(
			&guest.ID,
			&guest.EventID,
			&guest.CategoryID,
			&guest.Name,
			&guest.Email,
			&guest.PaidDate,
			&guest.TicketCount,
			&guest.CreatedAt,
			&guest.UpdatedAt,
			&guest.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan guest: %w", err)
		}
		out = append(out, guest)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list unbooked guests rows: %w", err)
	}

	return out, nil
}

func (r *GuestRepo) Update(ctx context.Context, db DBTX, g *model.Guest) (*model.Guest, error) {
	const query = `
		UPDATE guests SET
			category_id = $1,
			name = $2,
			email = $3,
			paid_date = $4,
			ticket_count = $5,
			updated_at = now()
		WHERE id = $6 AND deleted_at IS NULL
		RETURNING id, event_id, category_id, name, email, paid_date, ticket_count,
			created_at, updated_at, deleted_at
	`

	var guest model.Guest
	err := db.QueryRow(ctx, query,
		g.CategoryID,
		g.Name,
		g.Email,
		g.PaidDate,
		g.TicketCount,
		g.ID,
	).Scan(
		&guest.ID,
		&guest.EventID,
		&guest.CategoryID,
		&guest.Name,
		&guest.Email,
		&guest.PaidDate,
		&guest.TicketCount,
		&guest.CreatedAt,
		&guest.UpdatedAt,
		&guest.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update guest: %w", err)
	}

	return &guest, nil
}
