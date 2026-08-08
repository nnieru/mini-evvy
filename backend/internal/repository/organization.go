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

type OrganizationRepo struct {
	pool *pgxpool.Pool
}

func NewOrganizationRepo(pool *pgxpool.Pool) *OrganizationRepo {
	return &OrganizationRepo{pool: pool}
}

func (r *OrganizationRepo) Create(ctx context.Context, db DBTX, name string, ownerID uuid.UUID) (*model.Organization, error) {
	const query = `
		INSERT INTO organizations (name, status, owner_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, status, owner_id, created_at, updated_at, deleted_at
	`

	var organization model.Organization
	err := db.QueryRow(ctx, query, name, model.OrganizationActive, ownerID).Scan(
		&organization.ID,
		&organization.Name,
		&organization.Status,
		&organization.OwnerID,
		&organization.CreatedAt,
		&organization.UpdatedAt,
		&organization.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}

	return &organization, nil
}

func (r *OrganizationRepo) GetByID(ctx context.Context, db DBTX, id uuid.UUID) (*model.Organization, error) {
	const query = `
		SELECT id, name, status, owner_id, created_at, updated_at, deleted_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL
	`

	var organization model.Organization
	err := db.QueryRow(ctx, query, id).Scan(
		&organization.ID,
		&organization.Name,
		&organization.Status,
		&organization.OwnerID,
		&organization.CreatedAt,
		&organization.UpdatedAt,
		&organization.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get organization by id: %w", err)
	}

	return &organization, nil
}

func (r *OrganizationRepo) ListUserByID(ctx context.Context, db DBTX, userID uuid.UUID) ([]model.OrganizationWithRole, error) {
	const query = `
		SELECT o.id, o.name, o.status, o.owner_id, o.created_at, o.updated_at, o.deleted_at, r.name
		FROM organizations o
		INNER JOIN user_roles ur ON ur.organization_id = o.id
		INNER JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND o.deleted_at IS NULL
		ORDER BY o.created_at DESC
	`

	rows, err := db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list organizations by user id: %w", err)
	}
	defer rows.Close()

	var out []model.OrganizationWithRole
	for rows.Next() {
		var item model.OrganizationWithRole
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Status,
			&item.OwnerID,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.DeletedAt,
			&item.MyRole,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}

	return out, rows.Err()
}
