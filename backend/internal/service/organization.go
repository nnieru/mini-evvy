package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
)

var (
	ErrOrgNotFound   = errors.New("organization not found")
	ErrForbidden     = errors.New("forbidden")
	ErrUserNotFound  = errors.New("user not found")
	ErrAlreadyMember = errors.New("user is already a member of the organization")
	ErrInvalidRole   = errors.New("invalid role")
)

type orgStore interface {
	Create(ctx context.Context, db repository.DBTX, name string, ownerID uuid.UUID) (*model.Organization, error)
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.Organization, error)
	ListUserByID(ctx context.Context, db repository.DBTX, userID uuid.UUID) ([]model.OrganizationWithRole, error)
}

type roleStore interface {
	GetByName(ctx context.Context, db repository.DBTX, name string) (*model.Role, error)
}

type membershipStore interface {
	Create(ctx context.Context, db repository.DBTX, userID, roleID, orgID uuid.UUID) (*model.UserRole, error)
	IsMember(ctx context.Context, db repository.DBTX, userID, orgID uuid.UUID) (bool, error)
	ListByOrgID(ctx context.Context, db repository.DBTX, orgID uuid.UUID) ([]model.Member, error)
	HasRole(ctx context.Context, db repository.DBTX, userID, orgID uuid.UUID, roleNames ...string) (bool, error)
	GetRoleName(ctx context.Context, db repository.DBTX, userID, orgID uuid.UUID) (string, error)
}

type userLookup interface {
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

type OrganizationService struct {
	pool        *pgxpool.Pool
	orgs        orgStore
	roles       roleStore
	memberships membershipStore
	users       userLookup
}

func NewOrganizationService(
	pool *pgxpool.Pool,
	orgs orgStore,
	roles roleStore,
	memberships membershipStore,
	users userLookup,
) *OrganizationService {
	return &OrganizationService{
		pool:        pool,
		orgs:        orgs,
		roles:       roles,
		memberships: memberships,
		users:       users,
	}
}

func (s *OrganizationService) Create(ctx context.Context, ownerID uuid.UUID, name string) (*model.Organization, error) {
	memberships, err := s.orgs.ListUserByID(ctx, s.pool, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	if len(memberships) > 0 {
		hasStaff := false
		for _, m := range memberships {
			if m.MyRole == model.RoleOwner || m.MyRole == model.RoleAdmin {
				hasStaff = true
				break
			}
		}
		if !hasStaff {
			return nil, ErrForbidden
		}
	}

	ownerRole, err := s.roles.GetByName(ctx, s.pool, model.RoleOwner)
	if err != nil {
		return nil, fmt.Errorf("owner role missing — run EnsureDefaults: %w", err)
	}

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx)

	org, err := s.orgs.Create(ctx, tx, name, ownerID)
	if err != nil {
		return nil, err
	}

	if _, err := s.memberships.Create(ctx, tx, ownerID, ownerRole.ID, org.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return org, nil

}

func (s *OrganizationService) ListMine(ctx context.Context, userID uuid.UUID) ([]model.OrganizationWithRole, error) {
	return s.orgs.ListUserByID(ctx, s.pool, userID)
}

func (s *OrganizationService) Get(ctx context.Context, userID, orgID uuid.UUID) (*model.Organization, error) {
	ok, err := s.memberships.IsMember(ctx, s.pool, userID, orgID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	org, err := s.orgs.GetByID(ctx, s.pool, orgID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrOrgNotFound
	}

	return org, nil
}

func (s *OrganizationService) ListMembers(ctx context.Context, actorID, orgID uuid.UUID) ([]model.Member, error) {
	ok, err := s.memberships.IsMember(ctx, s.pool, actorID, orgID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}

	return s.memberships.ListByOrgID(ctx, s.pool, orgID)
}

func (s *OrganizationService) GetMyRole(ctx context.Context, userID, orgID uuid.UUID) (string, error) {
	roleName, err := s.memberships.GetRoleName(ctx, s.pool, userID, orgID)
	if errors.Is(err, repository.ErrNotFound) {
		return "", ErrForbidden
	}
	if err != nil {
		return "", err
	}
	return roleName, nil
}

func (s *OrganizationService) AddMember(ctx context.Context, actorID, orgID uuid.UUID, email string, roleName string) (*model.Member, error) {
	can, err := s.memberships.HasRole(ctx, s.pool, actorID, orgID, model.RoleOwner, model.RoleAdmin)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, ErrForbidden
	}

	if roleName != model.RoleAdmin && roleName != model.RoleMember {
		return nil, ErrInvalidRole
	}

	role, err := s.roles.GetByName(ctx, s.pool, roleName)
	if err != nil {
		return nil, err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	exists, err := s.memberships.IsMember(ctx, s.pool, user.ID, orgID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyMember
	}

	ur, err := s.memberships.Create(ctx, s.pool, user.ID, role.ID, orgID)
	if err != nil {
		return nil, err
	}

	return &model.Member{
		UserRoleID:     ur.ID,
		UserID:         user.ID,
		Name:           user.Name,
		Email:          user.Email,
		RoleID:         role.ID,
		RoleName:       role.Name,
		OrganizationID: orgID,
		CreatedAt:      ur.CreatedAt,
	}, nil
}
