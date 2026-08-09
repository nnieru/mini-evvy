package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
)

type SeatRepo struct {
	pool *pgxpool.Pool
}

func NewSeatRepo(pool *pgxpool.Pool) *SeatRepo {
	return &SeatRepo{pool: pool}
}

func (r *SeatRepo) CreateBatch(ctx context.Context, db DBTX, seats []model.Seat) ([]model.Seat, error) {
	const query = `
		INSERT INTO seats (
			code, section, "row", col, status, description, event_id, category_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, code, section, "row", col, status, description,
			event_id, category_id, created_at, updated_at, deleted_at
	`

	out := make([]model.Seat, 0, len(seats))
	for _, s := range seats {
		var seat model.Seat
		err := db.QueryRow(ctx, query,
			s.Code,
			s.Section,
			s.Row,
			s.Col,
			s.Status,
			s.Description,
			s.EventID,
			s.CategoryID,
		).Scan(
			&seat.ID,
			&seat.Code,
			&seat.Section,
			&seat.Row,
			&seat.Col,
			&seat.Status,
			&seat.Description,
			&seat.EventID,
			&seat.CategoryID,
			&seat.CreatedAt,
			&seat.UpdatedAt,
			&seat.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("create seat %s: %w", s.Code, err)
		}
		out = append(out, seat)
	}

	return out, nil
}

func (r *SeatRepo) GetByID(ctx context.Context, db DBTX, id uuid.UUID) (*model.Seat, error) {
	const query = `
		SELECT id, code, section, "row", col, status, description,
			event_id, category_id, created_at, updated_at, deleted_at
		FROM seats
		WHERE id = $1 AND deleted_at IS NULL
	`

	var seat model.Seat
	err := db.QueryRow(ctx, query, id).Scan(
		&seat.ID,
		&seat.Code,
		&seat.Section,
		&seat.Row,
		&seat.Col,
		&seat.Status,
		&seat.Description,
		&seat.EventID,
		&seat.CategoryID,
		&seat.CreatedAt,
		&seat.UpdatedAt,
		&seat.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get seat by id: %w", err)
	}

	return &seat, nil
}

type SeatListQuery struct {
	Status     *model.SeatStatus
	CategoryID *uuid.UUID
	PageQuery
}

func (r *SeatRepo) seatListWhere(eventID uuid.UUID, q SeatListQuery) (string, []any) {
	where := ` FROM seats WHERE event_id = $1 AND deleted_at IS NULL`
	args := []any{eventID}
	argPos := 2

	if q.Status != nil {
		where += fmt.Sprintf(` AND status = $%d`, argPos)
		args = append(args, *q.Status)
		argPos = len(args) + 1
	}
	if q.CategoryID != nil {
		where += fmt.Sprintf(` AND category_id = $%d`, argPos)
		args = append(args, *q.CategoryID)
		argPos = len(args) + 1
	}
	if strings.TrimSpace(q.Q) != "" {
		where += fmt.Sprintf(` AND (code ILIKE $%d OR COALESCE(section, '') ILIKE $%d)`, argPos, argPos)
		args = append(args, SearchPattern(q.Q))
	}

	return where, args
}

func (r *SeatRepo) CountByEventIDFiltered(ctx context.Context, db DBTX, eventID uuid.UUID, q SeatListQuery) (int, error) {
	where, args := r.seatListWhere(eventID, q)
	var total int
	if err := db.QueryRow(ctx, `SELECT COUNT(*)`+where, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count seats by event id: %w", err)
	}
	return total, nil
}

func (r *SeatRepo) ListByEventIDPaged(ctx context.Context, db DBTX, eventID uuid.UUID, q SeatListQuery) ([]model.Seat, error) {
	where, args := r.seatListWhere(eventID, q)
	offset := (q.Page - 1) * q.PageSize
	limitPos := len(args) + 1
	offsetPos := len(args) + 2
	query := `SELECT id, code, section, "row", col, status, description,
		event_id, category_id, created_at, updated_at, deleted_at` + where +
		fmt.Sprintf(` ORDER BY section NULLS LAST, "row" NULLS LAST, col NULLS LAST, code ASC LIMIT $%d OFFSET $%d`, limitPos, offsetPos)
	args = append(args, q.PageSize, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list seats by event id paged: %w", err)
	}
	defer rows.Close()

	var out []model.Seat
	for rows.Next() {
		var seat model.Seat
		if err := rows.Scan(
			&seat.ID,
			&seat.Code,
			&seat.Section,
			&seat.Row,
			&seat.Col,
			&seat.Status,
			&seat.Description,
			&seat.EventID,
			&seat.CategoryID,
			&seat.CreatedAt,
			&seat.UpdatedAt,
			&seat.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan seat: %w", err)
		}
		out = append(out, seat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list seats paged rows: %w", err)
	}

	return out, nil
}

func (r *SeatRepo) ListByEventID(ctx context.Context, db DBTX, eventID uuid.UUID, status *model.SeatStatus, categoryID *uuid.UUID) ([]model.Seat, error) {
	query := `
		SELECT id, code, section, "row", col, status, description,
			event_id, category_id, created_at, updated_at, deleted_at
		FROM seats
		WHERE event_id = $1 AND deleted_at IS NULL
	`
	args := []any{eventID}
	argPos := 2

	if status != nil {
		query += fmt.Sprintf(` AND status = $%d`, argPos)
		args = append(args, *status)
		argPos++
	}
	if categoryID != nil {
		query += fmt.Sprintf(` AND category_id = $%d`, argPos)
		args = append(args, *categoryID)
	}

	query += ` ORDER BY section NULLS LAST, "row" NULLS LAST, col NULLS LAST, code ASC`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list seats by event id: %w", err)
	}
	defer rows.Close()

	var out []model.Seat
	for rows.Next() {
		var seat model.Seat
		if err := rows.Scan(
			&seat.ID,
			&seat.Code,
			&seat.Section,
			&seat.Row,
			&seat.Col,
			&seat.Status,
			&seat.Description,
			&seat.EventID,
			&seat.CategoryID,
			&seat.CreatedAt,
			&seat.UpdatedAt,
			&seat.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan seat: %w", err)
		}
		out = append(out, seat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list seats rows: %w", err)
	}

	return out, nil
}

func (r *SeatRepo) Update(ctx context.Context, db DBTX, s *model.Seat) (*model.Seat, error) {
	const query = `
		UPDATE seats SET
			code = $1,
			section = $2,
			"row" = $3,
			col = $4,
			status = $5,
			description = $6,
			category_id = $7,
			updated_at = now()
		WHERE id = $8 AND deleted_at IS NULL
		RETURNING id, code, section, "row", col, status, description,
			event_id, category_id, created_at, updated_at, deleted_at
	`

	var seat model.Seat
	err := db.QueryRow(ctx, query,
		s.Code,
		s.Section,
		s.Row,
		s.Col,
		s.Status,
		s.Description,
		s.CategoryID,
		s.ID,
	).Scan(
		&seat.ID,
		&seat.Code,
		&seat.Section,
		&seat.Row,
		&seat.Col,
		&seat.Status,
		&seat.Description,
		&seat.EventID,
		&seat.CategoryID,
		&seat.CreatedAt,
		&seat.UpdatedAt,
		&seat.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update seat: %w", err)
	}

	return &seat, nil
}

func (r *SeatRepo) UpdateStatus(ctx context.Context, db DBTX, seatID uuid.UUID, status model.SeatStatus) error {
	const query = `
		UPDATE seats SET status = $1, updated_at = now()
		WHERE id = $2 AND deleted_at IS NULL
	`

	tag, err := db.Exec(ctx, query, status, seatID)
	if err != nil {
		return fmt.Errorf("update seat status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SeatRepo) ClaimFromAvailable(ctx context.Context, db DBTX, seatID uuid.UUID, status model.SeatStatus) error {
	const query = `
		UPDATE seats SET status = $1, updated_at = now()
		WHERE id = $2 AND status = $3 AND deleted_at IS NULL
	`

	tag, err := db.Exec(ctx, query, status, seatID, model.SeatAvailable)
	if err != nil {
		return fmt.Errorf("claim seat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSeatNotAvailable
	}
	return nil
}
