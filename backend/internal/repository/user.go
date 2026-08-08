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

var (
	ErrNotFound         = errors.New("not found")
	ErrSeatNotAvailable = errors.New("seat not available")
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, name, email, passwordHash string) (*model.User, error) {
	const query = `INSERT INTO users (name, email, password, status)
	VALUES ($1, $2, $3, $4)
	RETURNING id, name, email, status, password, created_at, updated_at, deleted_at`

	var user model.User
	err := r.pool.QueryRow(ctx, query, name, email, passwordHash, model.UserActive).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Status,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt)

	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &user, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	const query = `SELECT id, name, email, status, password, created_at, updated_at, deleted_at FROM users WHERE email = $1 AND deleted_at IS NULL`

	return r.scanOne(ctx, query, email)

}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	const query = `SELECT id, name, email, status, password, created_at, updated_at, deleted_at FROM users WHERE id = $1 AND deleted_at IS NULL`

	return r.scanOne(ctx, query, id)
}

func (r *UserRepo) scanOne(ctx context.Context, query string, args any) (*model.User, error) {

	var user model.User
	err := r.pool.QueryRow(ctx, query, args).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Status,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user %w", err)
	}

	return &user, nil
}
