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

type SeatCategoryRepo struct {
	pool *pgxpool.Pool
}

func NewSeatCategoryRepo(pool *pgxpool.Pool) *SeatCategoryRepo {
	return &SeatCategoryRepo{pool: pool}
}

func (r *SeatCategoryRepo) Create(ctx context.Context, db DBTX, c *model.SeatCategory) (*model.SeatCategory, error) {
	const query = `
		INSERT INTO seat_categories (name, code, price, currency, event_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, code, price, currency, event_id,
			created_at, updated_at, deleted_at
	`

	var sc model.SeatCategory
	err := db.QueryRow(ctx, query, c.Name, c.Code, c.Price, c.Currency, c.EventID).Scan(
		&sc.ID,
		&sc.Name,
		&sc.Code,
		&sc.Price,
		&sc.Currency,
		&sc.EventID,
		&sc.CreatedAt,
		&sc.UpdatedAt,
		&sc.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create seat category: %w", err)
	}

	return &sc, nil
}

func (r *SeatCategoryRepo) GetByID(ctx context.Context, db DBTX, id uuid.UUID) (*model.SeatCategory, error) {
	const query = `
		SELECT id, name, code, price, currency, event_id,
			created_at, updated_at, deleted_at
		FROM seat_categories
		WHERE id = $1 AND deleted_at IS NULL
	`

	var sc model.SeatCategory
	err := db.QueryRow(ctx, query, id).Scan(
		&sc.ID,
		&sc.Name,
		&sc.Code,
		&sc.Price,
		&sc.Currency,
		&sc.EventID,
		&sc.CreatedAt,
		&sc.UpdatedAt,
		&sc.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get seat category by id: %w", err)
	}

	return &sc, nil
}

func (r *SeatCategoryRepo) ListByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) ([]model.SeatCategory, error) {
	const query = `
		SELECT id, name, code, price, currency, event_id,
			created_at, updated_at, deleted_at
		FROM seat_categories
		WHERE event_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := db.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("list seat categories by event id: %w", err)
	}
	defer rows.Close()

	var out []model.SeatCategory
	for rows.Next() {
		var sc model.SeatCategory
		if err := rows.Scan(
			&sc.ID,
			&sc.Name,
			&sc.Code,
			&sc.Price,
			&sc.Currency,
			&sc.EventID,
			&sc.CreatedAt,
			&sc.UpdatedAt,
			&sc.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan seat category: %w", err)
		}
		out = append(out, sc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list seat categories rows: %w", err)
	}

	return out, nil
}

func (r *SeatCategoryRepo) Update(ctx context.Context, db DBTX, c *model.SeatCategory) (*model.SeatCategory, error) {
	const query = `
		UPDATE seat_categories SET
			name = $1,
			code = $2,
			price = $3,
			currency = $4,
			updated_at = now()
		WHERE id = $5 AND deleted_at IS NULL
		RETURNING id, name, code, price, currency, event_id,
			created_at, updated_at, deleted_at
	`

	var sc model.SeatCategory
	err := db.QueryRow(ctx, query, c.Name, c.Code, c.Price, c.Currency, c.ID).Scan(
		&sc.ID,
		&sc.Name,
		&sc.Code,
		&sc.Price,
		&sc.Currency,
		&sc.EventID,
		&sc.CreatedAt,
		&sc.UpdatedAt,
		&sc.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update seat category: %w", err)
	}

	return &sc, nil
}
