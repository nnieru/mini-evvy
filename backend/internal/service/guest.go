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
	ErrGuestNotFound = errors.New("guest not found")
)

type guestStore interface {
	Create(ctx context.Context, db repository.DBTX, g *model.Guest) (*model.Guest, error)
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.Guest, error)
	GetByEventNameEmailCategory(ctx context.Context, db repository.DBTX, eventID, categoryID uuid.UUID, name, email string) (*model.Guest, error)
	ListByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]model.Guest, error)
	Update(ctx context.Context, db repository.DBTX, g *model.Guest) (*model.Guest, error)
}

type GuestService struct {
	pool        *pgxpool.Pool
	guests      guestStore
	events      eventLookup
	categories  categoryLookup
	memberships membershipChecker
}

func NewGuestService(
	pool *pgxpool.Pool,
	guests guestStore,
	events eventLookup,
	categories categoryLookup,
	memberships membershipChecker,
) *GuestService {
	return &GuestService{
		pool:        pool,
		guests:      guests,
		events:      events,
		categories:  categories,
		memberships: memberships,
	}
}

type CreateGuestInput struct {
	Name        string
	Email       string
	CategoryID  uuid.UUID
	TicketCount int
	PaidDate    *time.Time
}

type UpdateGuestInput struct {
	Name        string
	Email       string
	CategoryID  uuid.UUID
	TicketCount int
	PaidDate    *time.Time
}

func (s *GuestService) validateCategoryForEvent(ctx context.Context, db repository.DBTX, eventID, categoryID uuid.UUID) error {
	category, err := s.categories.GetByID(ctx, db, categoryID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrCategoryNotFound
	}
	if err != nil {
		return fmt.Errorf("get category: %w", err)
	}
	if category.EventID != eventID {
		return ErrCategoryNotFound
	}
	return nil
}

func (s *GuestService) Create(ctx context.Context, actorID, eventID uuid.UUID, in CreateGuestInput) (*model.Guest, error) {
	event, err := s.events.GetByID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	can, err := s.memberships.IsMember(ctx, s.pool, actorID, event.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !can {
		return nil, ErrForbidden
	}

	if err := ensureSeatingEditable(event.SeatingPhase); err != nil {
		return nil, err
	}

	if in.Name == "" {
		return nil, fmt.Errorf("name required")
	}
	if in.Email == "" {
		return nil, fmt.Errorf("email required")
	}
	if in.CategoryID == uuid.Nil {
		return nil, fmt.Errorf("category_id required")
	}

	ticketCount := in.TicketCount
	if ticketCount == 0 {
		ticketCount = 1
	}
	if ticketCount < 1 {
		return nil, fmt.Errorf("ticket_count must be greater than 0")
	}

	if err := s.validateCategoryForEvent(ctx, s.pool, eventID, in.CategoryID); err != nil {
		return nil, err
	}

	guest := &model.Guest{
		EventID:     eventID,
		CategoryID:  in.CategoryID,
		Name:        in.Name,
		Email:       in.Email,
		PaidDate:    in.PaidDate,
		TicketCount: ticketCount,
	}

	created, err := s.guests.Create(ctx, s.pool, guest)
	if err != nil {
		return nil, fmt.Errorf("create guest: %w", err)
	}
	return created, nil
}

func (s *GuestService) ListByEvent(ctx context.Context, actorID, eventID uuid.UUID) ([]model.Guest, error) {
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

	list, err := s.guests.ListByEventID(ctx, s.pool, eventID)
	if err != nil {
		return nil, fmt.Errorf("list guests: %w", err)
	}
	return list, nil
}

func (s *GuestService) Get(ctx context.Context, actorID, guestID uuid.UUID) (*model.Guest, error) {
	guest, err := s.guests.GetByID(ctx, s.pool, guestID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrGuestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get guest: %w", err)
	}

	event, err := s.events.GetByID(ctx, s.pool, guest.EventID)
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

	return guest, nil
}

func (s *GuestService) Update(ctx context.Context, actorID, guestID uuid.UUID, in UpdateGuestInput) (*model.Guest, error) {
	existing, err := s.guests.GetByID(ctx, s.pool, guestID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrGuestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get guest: %w", err)
	}

	event, err := s.events.GetByID(ctx, s.pool, existing.EventID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	can, err := s.memberships.IsMember(ctx, s.pool, actorID, event.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !can {
		return nil, ErrForbidden
	}

	if err := ensureSeatingEditable(event.SeatingPhase); err != nil {
		return nil, err
	}

	if in.Name == "" {
		return nil, fmt.Errorf("name required")
	}
	if in.Email == "" {
		return nil, fmt.Errorf("email required")
	}
	if in.CategoryID == uuid.Nil {
		return nil, fmt.Errorf("category_id required")
	}
	if in.TicketCount < 1 {
		return nil, fmt.Errorf("ticket_count must be greater than 0")
	}

	if err := s.validateCategoryForEvent(ctx, s.pool, existing.EventID, in.CategoryID); err != nil {
		return nil, err
	}

	existing.CategoryID = in.CategoryID
	existing.Name = in.Name
	existing.Email = in.Email
	existing.PaidDate = in.PaidDate
	existing.TicketCount = in.TicketCount

	updated, err := s.guests.Update(ctx, s.pool, existing)
	if err != nil {
		return nil, fmt.Errorf("update guest: %w", err)
	}
	return updated, nil
}
