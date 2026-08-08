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
	ErrCategoryNotFound = errors.New("seat category not found")
)

type categoryStore interface {
	Create(ctx context.Context, db repository.DBTX, c *model.SeatCategory) (*model.SeatCategory, error)
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.SeatCategory, error)
	ListByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]model.SeatCategory, error)
	Update(ctx context.Context, db repository.DBTX, c *model.SeatCategory) (*model.SeatCategory, error)
}

type eventLookup interface {
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.Event, error)
}

type SeatCategoryService struct {
	pool        *pgxpool.Pool
	categories  categoryStore
	events      eventLookup
	memberships membershipChecker
}

func NewSeatCategoryService(
	pool *pgxpool.Pool,
	categories categoryStore,
	events eventLookup,
	memberships membershipChecker,
) *SeatCategoryService {
	return &SeatCategoryService{
		pool:        pool,
		categories:  categories,
		events:      events,
		memberships: memberships,
	}
}

type CreateCategoryInput struct {
	Name     string
	Code     *string
	Price    float64
	Currency string
}

type UpdateCategoryInput struct {
	Name     string
	Code     *string
	Price    float64
	Currency string
}

func (s *SeatCategoryService) Create(ctx context.Context, actorID, eventID uuid.UUID, in CreateCategoryInput) (*model.SeatCategory, error) {
	event, err := s.events.GetByID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	can, err := s.memberships.HasRole(ctx, s.pool, actorID, event.OrganizationID, model.RoleOwner, model.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if !can {
		return nil, ErrForbidden
	}

	if in.Name == "" {
		return nil, fmt.Errorf("name required")
	}
	if in.Price < 0 {
		return nil, fmt.Errorf("price must be non-negative")
	}

	currency := in.Currency
	if currency == "" {
		currency = "IDR"
	}

	category := &model.SeatCategory{
		Name:     in.Name,
		Code:     in.Code,
		Price:    in.Price,
		Currency: currency,
		EventID:  eventID,
	}

	created, err := s.categories.Create(ctx, s.pool, category)
	if err != nil {
		return nil, fmt.Errorf("create seat category: %w", err)
	}
	return created, nil
}

func (s *SeatCategoryService) ListByEvent(ctx context.Context, actorID, eventID uuid.UUID) ([]model.SeatCategory, error) {
	event, err := s.events.GetByID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	ok, err := s.memberships.IsMember(ctx, s.pool, actorID, event.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !ok {
		return nil, ErrForbidden
	}

	list, err := s.categories.ListByEventID(ctx, s.pool, eventID)
	if err != nil {
		return nil, fmt.Errorf("list seat categories: %w", err)
	}
	return list, nil
}

func (s *SeatCategoryService) Get(ctx context.Context, actorID, categoryID uuid.UUID) (*model.SeatCategory, error) {
	category, err := s.categories.GetByID(ctx, s.pool, categoryID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrCategoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get seat category: %w", err)
	}

	event, err := s.events.GetByID(ctx, s.pool, category.EventID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	ok, err := s.memberships.IsMember(ctx, s.pool, actorID, event.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !ok {
		return nil, ErrForbidden
	}

	return category, nil
}

func (s *SeatCategoryService) Update(ctx context.Context, actorID, categoryID uuid.UUID, in UpdateCategoryInput) (*model.SeatCategory, error) {
	existing, err := s.categories.GetByID(ctx, s.pool, categoryID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrCategoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get seat category: %w", err)
	}

	event, err := s.events.GetByID(ctx, s.pool, existing.EventID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	can, err := s.memberships.HasRole(ctx, s.pool, actorID, event.OrganizationID, model.RoleOwner, model.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if !can {
		return nil, ErrForbidden
	}

	if in.Name == "" {
		return nil, fmt.Errorf("name required")
	}
	if in.Price < 0 {
		return nil, fmt.Errorf("price must be non-negative")
	}

	currency := in.Currency
	if currency == "" {
		currency = "IDR"
	}

	existing.Name = in.Name
	existing.Code = in.Code
	existing.Price = in.Price
	existing.Currency = currency

	updated, err := s.categories.Update(ctx, s.pool, existing)
	if err != nil {
		return nil, fmt.Errorf("update seat category: %w", err)
	}
	return updated, nil
}
