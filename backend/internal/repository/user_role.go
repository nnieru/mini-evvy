package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/nnieru/mini-evvy/internal/model"
)

type UserRoleRepo struct{}

func NewUserRoleRepo() *UserRoleRepo {
	return &UserRoleRepo{}
}

func (r *UserRoleRepo) Create(ctx context.Context, db DBTX, userID, roleID, orgID uuid.UUID) (*model.UserRole, error) {
	const query = `INSERT INTO user_roles (user_id, role_id, organization_id) VALUES ($1, $2, $3) RETURNING id, user_id, role_id, organization_id, created_at, updated_at`

	var userRole model.UserRole
	err := db.QueryRow(ctx, query, userID, roleID, orgID).Scan(&userRole.ID, &userRole.UserID, &userRole.RoleID, &userRole.OrganizationID, &userRole.CreatedAt, &userRole.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("create user role: %w", err)
	}

	return &userRole, nil
}

func (r *UserRoleRepo) IsMember(ctx context.Context, db DBTX, userID, orgID uuid.UUID) (bool, error) {
	const query = `SELECT 1 FROM user_roles WHERE user_id = $1 AND organization_id = $2 LIMIT 1`

	var n int
	err := db.QueryRow(ctx, query, userID, orgID).Scan(&n)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("is member: %w", err)
	}

	return true, nil
}

func (r *UserRoleRepo) ListByOrgID(ctx context.Context, db DBTX, orgID uuid.UUID) ([]model.Member, error) {
	const query = `
		SELECT ur.id, u.id, u.name, u.email, r.id, r.name, ur.organization_id, ur.created_at
		FROM user_roles ur
		INNER JOIN users u ON u.id = ur.user_id
		INNER JOIN roles r ON r.id = ur.role_id
		WHERE ur.organization_id = $1 AND u.deleted_at IS NULL
		ORDER BY ur.created_at ASC
	`

	rows, err := db.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var out []model.Member
	for rows.Next() {
		var m model.Member
		if err := rows.Scan(
			&m.UserRoleID,
			&m.UserID,
			&m.Name,
			&m.Email,
			&m.RoleID,
			&m.RoleName,
			&m.OrganizationID,
			&m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		out = append(out, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list members rows: %w", err)
	}

	return out, nil
}

func (r *UserRoleRepo) HasRole(ctx context.Context, db DBTX, userID, orgID uuid.UUID, roleNames ...string) (bool, error) {
	const query = `SELECT 1
	FROM user_roles ur
	INNER JOIN roles r ON r.id = ur.role_id
	WHERE ur.user_id = $1 AND ur.organization_id = $2 AND r.name = ANY($3) LIMIT 1`

	var n int
	err := db.QueryRow(ctx, query, userID, orgID, roleNames).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *UserRoleRepo) GetRoleName(ctx context.Context, db DBTX, userID, orgID uuid.UUID) (string, error) {
	const query = `
		SELECT r.name
		FROM user_roles ur
		INNER JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND ur.organization_id = $2
		LIMIT 1
	`

	var roleName string
	err := db.QueryRow(ctx, query, userID, orgID).Scan(&roleName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get role name: %w", err)
	}
	return roleName, nil
}
