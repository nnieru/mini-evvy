package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
)

var (
	ErrEventNotFound = errors.New("event not found")
)

type eventStore interface {
	Create(ctx context.Context, db repository.DBTX, e *model.Event) (*model.Event, error)
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.Event, error)
	ListByOrgID(ctx context.Context, db repository.DBTX, orgID uuid.UUID) ([]model.Event, error)
	ListByUserID(ctx context.Context, db repository.DBTX, userID uuid.UUID) ([]model.EventWithOrganization, error)
	Update(ctx context.Context, db repository.DBTX, e *model.Event) (*model.Event, error)
}

type membershipChecker interface {
	IsMember(ctx context.Context, db repository.DBTX, userID, orgID uuid.UUID) (bool, error)
	HasRole(ctx context.Context, db repository.DBTX, userID, orgID uuid.UUID, roleNames ...string) (bool, error)
}

type EventService struct {
	pool        *pgxpool.Pool
	events      eventStore
	memberships membershipChecker
}

func NewEventService(pool *pgxpool.Pool, events eventStore, memberships membershipChecker) *EventService {
	return &EventService{
		pool:        pool,
		events:      events,
		memberships: memberships,
	}
}

type CreateEventInput struct {
	Name        string
	Description *string
	StartDate   time.Time
	EndDate     *time.Time
	StartTime   *string
	EndTime     *string
}

type UpdateEventInput struct {
	Name        string
	Status      model.EventStatus
	Description *string
	StartDate   time.Time
	EndDate     *time.Time
	StartTime   *string
	EndTime     *string
}

func (s *EventService) Create(ctx context.Context, actorID, orgID uuid.UUID, in CreateEventInput) (*model.Event, error) {
	can, err := s.memberships.HasRole(ctx, s.pool, actorID, orgID, model.RoleOwner, model.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if !can {
		return nil, ErrForbidden
	}

	if in.Name == "" {
		return nil, fmt.Errorf("name required")
	}
	if in.StartDate.IsZero() {
		return nil, fmt.Errorf("start_date required")
	}
	if in.EndDate != nil && in.EndDate.Before(in.StartDate) {
		return nil, fmt.Errorf("end_date must be on or after start_date")
	}

	event := &model.Event{
		Name:           in.Name,
		Status:         model.EventActive,
		Description:    in.Description,
		StartDate:      in.StartDate,
		EndDate:        in.EndDate,
		StartTime:      in.StartTime,
		EndTime:        in.EndTime,
		CreatorID:      actorID,
		OrganizationID: orgID,
	}

	created, err := s.events.Create(ctx, s.pool, event)
	if err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}
	return created, nil
}

func (s *EventService) ListByOrg(ctx context.Context, actorID, orgID uuid.UUID) ([]model.Event, error) {
	ok, err := s.memberships.IsMember(ctx, s.pool, actorID, orgID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !ok {
		return nil, ErrForbidden
	}

	list, err := s.events.ListByOrgID(ctx, s.pool, orgID)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return list, nil
}

func (s *EventService) ListMine(ctx context.Context, actorID uuid.UUID) ([]model.EventWithOrganization, error) {
	list, err := s.events.ListByUserID(ctx, s.pool, actorID)
	if err != nil {
		return nil, fmt.Errorf("list my events: %w", err)
	}
	return list, nil
}

func (s *EventService) Get(ctx context.Context, actorID, eventID uuid.UUID) (*model.Event, error) {
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

	return event, nil
}

func (s *EventService) Update(ctx context.Context, actorID, eventID uuid.UUID, in UpdateEventInput) (*model.Event, error) {
	existing, err := s.events.GetByID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	can, err := s.memberships.HasRole(ctx, s.pool, actorID, existing.OrganizationID, model.RoleOwner, model.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if !can {
		return nil, ErrForbidden
	}

	if in.Name == "" {
		return nil, fmt.Errorf("name required")
	}
	if in.StartDate.IsZero() {
		return nil, fmt.Errorf("start_date required")
	}
	if in.EndDate != nil && in.EndDate.Before(in.StartDate) {
		return nil, fmt.Errorf("end_date must be on or after start_date")
	}

	existing.Name = in.Name
	existing.Description = in.Description
	existing.StartDate = in.StartDate
	existing.EndDate = in.EndDate
	existing.StartTime = in.StartTime
	existing.EndTime = in.EndTime
	if in.Status != "" {
		existing.Status = in.Status
	}

	updated, err := s.events.Update(ctx, s.pool, existing)
	if err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}
	return updated, nil
}
