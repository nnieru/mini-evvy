package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
)

type RoleRepo struct {
	pool *pgxpool.Pool
}

func NewRoleRepo(pool *pgxpool.Pool) *RoleRepo {
	return &RoleRepo{pool: pool}
}

func (r *RoleRepo) EnsureDefaults(ctx context.Context) error {
	roles := []struct {
		Name string
		Desc string
	}{
		{model.RoleOwner, "Organization Owner"},
		{model.RoleAdmin, "Organization Admin"},
		{model.RoleMember, "Organization Member"},
	}

	for _, role := range roles {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO roles (name, description)
			VALUES ($1, $2)
			ON CONFLICT (name) DO NOTHING
		`, role.Name, role.Desc)
		if err != nil {
			return fmt.Errorf("seed role %s: %w", role.Name, err)
		}
	}

	return nil
}

func (r *RoleRepo) GetByName(ctx context.Context, db DBTX, name string) (*model.Role, error) {
	const query = `
		SELECT id, name, description, created_by, updated_by, created_at, updated_at, deleted_at
		FROM roles
		WHERE name = $1 AND deleted_at IS NULL
	`

	var role model.Role
	err := db.QueryRow(ctx, query, name).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.CreatedBy,
		&role.UpdatedBy,
		&role.CreatedAt,
		&role.UpdatedAt,
		&role.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}

	return &role, nil
}
