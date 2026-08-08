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

type EventRepo struct {
	pool *pgxpool.Pool
}

func NewEventRepo(pool *pgxpool.Pool) *EventRepo {
	return &EventRepo{pool: pool}
}

func (r *EventRepo) Create(ctx context.Context, db DBTX, e *model.Event) (*model.Event, error) {
	const query = `
		INSERT INTO events (
			name, status, description, start_date, end_date,
			start_time, end_time, creator_id, organization_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, status, seating_phase, description, start_date, end_date,
			start_time::text, end_time::text, creator_id, organization_id,
			created_at, updated_at, deleted_at
	`

	var event model.Event
	err := db.QueryRow(ctx, query,
		e.Name,
		e.Status,
		e.Description,
		e.StartDate,
		e.EndDate,
		e.StartTime,
		e.EndTime,
		e.CreatorID,
		e.OrganizationID,
	).Scan(
		&event.ID,
		&event.Name,
		&event.Status,
		&event.SeatingPhase,
		&event.Description,
		&event.StartDate,
		&event.EndDate,
		&event.StartTime,
		&event.EndTime,
		&event.CreatorID,
		&event.OrganizationID,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}

	return &event, nil
}

func (r *EventRepo) GetByID(ctx context.Context, db DBTX, id uuid.UUID) (*model.Event, error) {
	const query = `
		SELECT id, name, status, seating_phase, description, start_date, end_date,
			start_time::text, end_time::text, creator_id, organization_id,
			created_at, updated_at, deleted_at
		FROM events
		WHERE id = $1 AND deleted_at IS NULL
	`

	var event model.Event
	err := db.QueryRow(ctx, query, id).Scan(
		&event.ID,
		&event.Name,
		&event.Status,
		&event.SeatingPhase,
		&event.Description,
		&event.StartDate,
		&event.EndDate,
		&event.StartTime,
		&event.EndTime,
		&event.CreatorID,
		&event.OrganizationID,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event by id: %w", err)
	}

	return &event, nil
}

func (r *EventRepo) ListByOrgID(ctx context.Context, db DBTX, orgID uuid.UUID) ([]model.Event, error) {
	const query = `
		SELECT id, name, status, seating_phase, description, start_date, end_date,
			start_time::text, end_time::text, creator_id, organization_id,
			created_at, updated_at, deleted_at
		FROM events
		WHERE organization_id = $1 AND deleted_at IS NULL
		ORDER BY start_date ASC, created_at DESC
	`

	rows, err := db.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list events by org id: %w", err)
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var event model.Event
		if err := rows.Scan(
			&event.ID,
			&event.Name,
			&event.Status,
			&event.SeatingPhase,
			&event.Description,
			&event.StartDate,
			&event.EndDate,
			&event.StartTime,
			&event.EndTime,
			&event.CreatorID,
			&event.OrganizationID,
			&event.CreatedAt,
			&event.UpdatedAt,
			&event.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list events rows: %w", err)
	}

	return events, nil
}

func (r *EventRepo) ListByUserID(ctx context.Context, db DBTX, userID uuid.UUID) ([]model.EventWithOrganization, error) {
	const query = `
		SELECT e.id, e.name, e.status, e.seating_phase, e.description, e.start_date, e.end_date,
			e.start_time::text, e.end_time::text, e.creator_id, e.organization_id,
			e.created_at, e.updated_at, e.deleted_at, o.name
		FROM events e
		INNER JOIN user_roles ur ON ur.organization_id = e.organization_id
		INNER JOIN organizations o ON o.id = e.organization_id
		WHERE ur.user_id = $1 AND e.deleted_at IS NULL AND o.deleted_at IS NULL
		ORDER BY e.start_date ASC, e.created_at DESC
	`

	rows, err := db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list events by user id: %w", err)
	}
	defer rows.Close()

	var events []model.EventWithOrganization
	for rows.Next() {
		var item model.EventWithOrganization
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Status,
			&item.SeatingPhase,
			&item.Description,
			&item.StartDate,
			&item.EndDate,
			&item.StartTime,
			&item.EndTime,
			&item.CreatorID,
			&item.OrganizationID,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.DeletedAt,
			&item.OrganizationName,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list events rows: %w", err)
	}

	return events, nil
}

func (r *EventRepo) Update(ctx context.Context, db DBTX, e *model.Event) (*model.Event, error) {
	const query = `
		UPDATE events SET
			name = $1,
			status = $2,
			description = $3,
			start_date = $4,
			end_date = $5,
			start_time = $6,
			end_time = $7,
			updated_at = now()
		WHERE id = $8 AND deleted_at IS NULL
		RETURNING id, name, status, seating_phase, description, start_date, end_date,
			start_time::text, end_time::text, creator_id, organization_id,
			created_at, updated_at, deleted_at
	`

	var event model.Event
	err := db.QueryRow(ctx, query,
		e.Name,
		e.Status,
		e.Description,
		e.StartDate,
		e.EndDate,
		e.StartTime,
		e.EndTime,
		e.ID,
	).Scan(
		&event.ID,
		&event.Name,
		&event.Status,
		&event.SeatingPhase,
		&event.Description,
		&event.StartDate,
		&event.EndDate,
		&event.StartTime,
		&event.EndTime,
		&event.CreatorID,
		&event.OrganizationID,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	return &event, nil
}

func (r *EventRepo) UpdateSeatingPhase(ctx context.Context, db DBTX, eventID uuid.UUID, phase model.SeatingPhase) error {
	const query = `
		UPDATE events SET seating_phase = $1, updated_at = now()
		WHERE id = $2 AND deleted_at IS NULL
	`

	tag, err := db.Exec(ctx, query, phase, eventID)
	if err != nil {
		return fmt.Errorf("update seating phase: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
