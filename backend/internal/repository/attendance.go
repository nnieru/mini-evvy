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

type AttendanceRepo struct {
	pool *pgxpool.Pool
}

func NewAttendanceRepo(pool *pgxpool.Pool) *AttendanceRepo {
	return &AttendanceRepo{pool: pool}
}

func (r *AttendanceRepo) Create(ctx context.Context, db DBTX, a *model.AttendanceLog) (*model.AttendanceLog, error) {
	const query = `
		INSERT INTO attendance_logs (
			guest_id, event_id, seat_id, status, message, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, guest_id, event_id, seat_id, status, message,
			created_by, updated_by, created_at, updated_at, deleted_at
	`

	var log model.AttendanceLog
	err := db.QueryRow(ctx, query,
		a.GuestID,
		a.EventID,
		a.SeatID,
		a.Status,
		a.Message,
		a.CreatedBy,
		a.UpdatedBy,
	).Scan(
		&log.ID,
		&log.GuestID,
		&log.EventID,
		&log.SeatID,
		&log.Status,
		&log.Message,
		&log.CreatedBy,
		&log.UpdatedBy,
		&log.CreatedAt,
		&log.UpdatedAt,
		&log.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create attendance log: %w", err)
	}

	return &log, nil
}

func (r *AttendanceRepo) ListByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) ([]model.AttendanceLog, error) {
	const query = `
		SELECT id, guest_id, event_id, seat_id, status, message,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM attendance_logs
		WHERE event_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := db.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("list attendance by event id: %w", err)
	}
	defer rows.Close()

	var out []model.AttendanceLog
	for rows.Next() {
		var log model.AttendanceLog
		if err := rows.Scan(
			&log.ID,
			&log.GuestID,
			&log.EventID,
			&log.SeatID,
			&log.Status,
			&log.Message,
			&log.CreatedBy,
			&log.UpdatedBy,
			&log.CreatedAt,
			&log.UpdatedAt,
			&log.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan attendance log: %w", err)
		}
		out = append(out, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list attendance rows: %w", err)
	}

	return out, nil
}

func (r *AttendanceRepo) GetByID(ctx context.Context, db DBTX, id uuid.UUID) (*model.AttendanceLog, error) {
	const query = `
		SELECT id, guest_id, event_id, seat_id, status, message,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM attendance_logs
		WHERE id = $1 AND deleted_at IS NULL
	`

	var log model.AttendanceLog
	err := db.QueryRow(ctx, query, id).Scan(
		&log.ID,
		&log.GuestID,
		&log.EventID,
		&log.SeatID,
		&log.Status,
		&log.Message,
		&log.CreatedBy,
		&log.UpdatedBy,
		&log.CreatedAt,
		&log.UpdatedAt,
		&log.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get attendance by id: %w", err)
	}

	return &log, nil
}

func (r *AttendanceRepo) Update(ctx context.Context, db DBTX, a *model.AttendanceLog) (*model.AttendanceLog, error) {
	const query = `
		UPDATE attendance_logs SET
			status = $1,
			message = $2,
			updated_by = $3,
			updated_at = now()
		WHERE id = $4 AND deleted_at IS NULL
		RETURNING id, guest_id, event_id, seat_id, status, message,
			created_by, updated_by, created_at, updated_at, deleted_at
	`

	var log model.AttendanceLog
	err := db.QueryRow(ctx, query,
		a.Status,
		a.Message,
		a.UpdatedBy,
		a.ID,
	).Scan(
		&log.ID,
		&log.GuestID,
		&log.EventID,
		&log.SeatID,
		&log.Status,
		&log.Message,
		&log.CreatedBy,
		&log.UpdatedBy,
		&log.CreatedAt,
		&log.UpdatedAt,
		&log.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update attendance log: %w", err)
	}

	return &log, nil
}

func (r *AttendanceRepo) SoftDelete(ctx context.Context, db DBTX, id, updatedBy uuid.UUID) error {
	const query = `
		UPDATE attendance_logs SET
			updated_by = $1,
			deleted_at = now(),
			updated_at = now()
		WHERE id = $2 AND deleted_at IS NULL
	`

	tag, err := db.Exec(ctx, query, updatedBy, id)
	if err != nil {
		return fmt.Errorf("soft delete attendance log: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *AttendanceRepo) ExistsCheckedIn(ctx context.Context, db DBTX, eventID, guestID, seatID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM attendance_logs
			WHERE event_id = $1 AND guest_id = $2 AND seat_id = $3
				AND status = 'checked_in' AND deleted_at IS NULL
		)
	`

	var exists bool
	err := db.QueryRow(ctx, query, eventID, guestID, seatID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("exists checked in: %w", err)
	}
	return exists, nil
}
