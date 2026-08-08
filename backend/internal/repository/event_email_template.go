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

type EventEmailTemplateRepo struct {
	pool *pgxpool.Pool
}

func NewEventEmailTemplateRepo(pool *pgxpool.Pool) *EventEmailTemplateRepo {
	return &EventEmailTemplateRepo{pool: pool}
}

func (r *EventEmailTemplateRepo) GetByEventAndType(ctx context.Context, db DBTX, eventID uuid.UUID, templateType string) (*model.EventEmailTemplate, error) {
	const query = `
		SELECT id, event_id, type, config, created_at, updated_at, updated_by
		FROM event_email_templates
		WHERE event_id = $1 AND type = $2
	`

	var t model.EventEmailTemplate
	err := db.QueryRow(ctx, query, eventID, templateType).Scan(
		&t.ID,
		&t.EventID,
		&t.Type,
		&t.Config,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.UpdatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event email template: %w", err)
	}

	return &t, nil
}

func (r *EventEmailTemplateRepo) Upsert(ctx context.Context, db DBTX, t *model.EventEmailTemplate) (*model.EventEmailTemplate, error) {
	const query = `
		INSERT INTO event_email_templates (event_id, type, config, updated_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_id, type) DO UPDATE SET
			config = EXCLUDED.config,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()
		RETURNING id, event_id, type, config, created_at, updated_at, updated_by
	`

	var out model.EventEmailTemplate
	err := db.QueryRow(ctx, query, t.EventID, t.Type, t.Config, t.UpdatedBy).Scan(
		&out.ID,
		&out.EventID,
		&out.Type,
		&out.Config,
		&out.CreatedAt,
		&out.UpdatedAt,
		&out.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert event email template: %w", err)
	}

	return &out, nil
}

func (r *EventEmailTemplateRepo) DeleteByEventAndType(ctx context.Context, db DBTX, eventID uuid.UUID, templateType string) error {
	const query = `
		DELETE FROM event_email_templates
		WHERE event_id = $1 AND type = $2
	`

	tag, err := db.Exec(ctx, query, eventID, templateType)
	if err != nil {
		return fmt.Errorf("delete event email template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
