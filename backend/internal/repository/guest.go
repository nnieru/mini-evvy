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

func (r *GuestRepo) GetSoftDeletedByEventNameEmailCategory(
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
			AND deleted_at IS NOT NULL
		ORDER BY deleted_at DESC
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
		return nil, fmt.Errorf("get soft deleted guest by event name email category: %w", err)
	}

	return &guest, nil
}

func (r *GuestRepo) RestoreSoftDeleted(ctx context.Context, db DBTX, g *model.Guest) (*model.Guest, error) {
	const query = `
		UPDATE guests SET
			category_id = $1,
			name = $2,
			email = $3,
			paid_date = $4,
			ticket_count = $5,
			deleted_at = NULL,
			updated_at = now()
		WHERE id = $6 AND deleted_at IS NOT NULL
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
		return nil, fmt.Errorf("restore soft deleted guest: %w", err)
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

func (r *GuestRepo) CountByEventIDFiltered(ctx context.Context, db DBTX, eventID uuid.UUID, q string) (int, error) {
	args := []any{eventID}
	where := ` FROM guests WHERE event_id = $1 AND deleted_at IS NULL`
	where, args = appendILikeOr(where, []string{"name", "email"}, q, args)

	var total int
	if err := db.QueryRow(ctx, `SELECT COUNT(*)`+where, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count guests by event id: %w", err)
	}
	return total, nil
}

func (r *GuestRepo) ListByEventIDPaged(ctx context.Context, db DBTX, eventID uuid.UUID, pq PageQuery) ([]model.Guest, error) {
	args := []any{eventID}
	where := ` FROM guests WHERE event_id = $1 AND deleted_at IS NULL`
	where, args = appendILikeOr(where, []string{"name", "email"}, pq.Q, args)

	offset := (pq.Page - 1) * pq.PageSize
	limitPos := len(args) + 1
	offsetPos := len(args) + 2
	query := `SELECT id, event_id, category_id, name, email, paid_date, ticket_count,
		created_at, updated_at, deleted_at` + where +
		fmt.Sprintf(` ORDER BY created_at ASC LIMIT $%d OFFSET $%d`, limitPos, offsetPos)
	args = append(args, pq.PageSize, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list guests by event id paged: %w", err)
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
		return nil, fmt.Errorf("list guests paged rows: %w", err)
	}

	return out, nil
}

const unbookedGuestsWhere = `
		FROM guests g
		WHERE g.event_id = $1 AND g.deleted_at IS NULL
			AND (
				SELECT COUNT(*) FROM seat_bookings sb
				WHERE sb.guest_id = g.id AND sb.deleted_at IS NULL AND sb.status <> 'cancelled'
			) < g.ticket_count
`

func (r *GuestRepo) CountUnbookedByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*)` + unbookedGuestsWhere
	var total int
	if err := db.QueryRow(ctx, query, eventID).Scan(&total); err != nil {
		return 0, fmt.Errorf("count unbooked guests by event id: %w", err)
	}
	return total, nil
}

func (r *GuestRepo) ListUnbookedByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) ([]model.Guest, error) {
	query := `
		SELECT g.id, g.event_id, g.category_id, g.name, g.email, g.paid_date, g.ticket_count,
			g.created_at, g.updated_at, g.deleted_at
` + unbookedGuestsWhere + `
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

func (r *GuestRepo) SoftDeleteUnbookedByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) error {
	const query = `
		UPDATE guests
		SET deleted_at = now(), updated_at = now()
		WHERE event_id = $1
			AND deleted_at IS NULL
			AND NOT EXISTS (
				SELECT 1 FROM seat_bookings sb
				WHERE sb.guest_id = guests.id
					AND sb.deleted_at IS NULL
					AND sb.status <> 'cancelled'
			)
	`
	if _, err := db.Exec(ctx, query, eventID); err != nil {
		return fmt.Errorf("soft delete unbooked guests: %w", err)
	}
	return nil
}

func (r *GuestRepo) SoftDeleteAllByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) error {
	const query = `
		UPDATE guests
		SET deleted_at = now(), updated_at = now()
		WHERE event_id = $1 AND deleted_at IS NULL
	`
	if _, err := db.Exec(ctx, query, eventID); err != nil {
		return fmt.Errorf("soft delete all guests: %w", err)
	}
	return nil
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
