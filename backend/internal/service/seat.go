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
	ErrSeatNotFound = errors.New("seat not found")
	ErrInvalidSeatStatus = errors.New("invalid seat status")
)

type seatStore interface {
	CreateBatch(ctx context.Context, db repository.DBTX, seats []model.Seat) ([]model.Seat, error)
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.Seat, error)
	ListByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID, status *model.SeatStatus, categoryID *uuid.UUID) ([]model.Seat, error)
	Update(ctx context.Context, db repository.DBTX, s *model.Seat) (*model.Seat, error)
}

type categoryLookup interface {
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.SeatCategory, error)
	ListByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]model.SeatCategory, error)
}

type SeatService struct {
	pool        *pgxpool.Pool
	seats       seatStore
	events      eventLookup
	categories  categoryLookup
	memberships membershipChecker
}

func NewSeatService(
	pool *pgxpool.Pool,
	seats seatStore,
	events eventLookup,
	categories categoryLookup,
	memberships membershipChecker,
) *SeatService {
	return &SeatService{
		pool:        pool,
		seats:       seats,
		events:      events,
		categories:  categories,
		memberships: memberships,
	}
}

type CreateSeatInput struct {
	Code        string
	CategoryID  uuid.UUID
	Section     *string
	Row         *int
	Col         *int
	Description *string
	Status      model.SeatStatus
}

type UpdateSeatInput struct {
	Code        string
	CategoryID  uuid.UUID
	Section     *string
	Row         *int
	Col         *int
	Description *string
	Status      model.SeatStatus
}

func isValidSeatStatus(status model.SeatStatus) bool {
	switch status {
	case model.SeatAvailable, model.SeatReserved, model.SeatOccupied, model.SeatBlocked:
		return true
	default:
		return false
	}
}

func (s *SeatService) validateCategoryForEvent(ctx context.Context, db repository.DBTX, eventID, categoryID uuid.UUID) error {
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

func (s *SeatService) CreateBatch(ctx context.Context, actorID, eventID uuid.UUID, inputs []CreateSeatInput) ([]model.Seat, error) {
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

	if err := ensureSeatingEditable(event.SeatingPhase); err != nil {
		return nil, err
	}

	if len(inputs) == 0 {
		return nil, fmt.Errorf("at least one seat required")
	}

	seats := make([]model.Seat, 0, len(inputs))
	for _, in := range inputs {
		if in.Code == "" {
			return nil, fmt.Errorf("code required")
		}
		if in.CategoryID == uuid.Nil {
			return nil, fmt.Errorf("category_id required")
		}

		status := in.Status
		if status == "" {
			status = model.SeatAvailable
		}
		if !isValidSeatStatus(status) {
			return nil, ErrInvalidSeatStatus
		}

		if err := s.validateCategoryForEvent(ctx, s.pool, eventID, in.CategoryID); err != nil {
			return nil, err
		}

		seats = append(seats, model.Seat{
			Code:        in.Code,
			Section:     in.Section,
			Row:         in.Row,
			Col:         in.Col,
			Status:      status,
			Description: in.Description,
			EventID:     eventID,
			CategoryID:  in.CategoryID,
		})
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := s.seats.CreateBatch(ctx, tx, seats)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}

func (s *SeatService) ListByEvent(ctx context.Context, actorID, eventID uuid.UUID, statusFilter *model.SeatStatus, categoryID *uuid.UUID) ([]model.Seat, error) {
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

	if statusFilter != nil && !isValidSeatStatus(*statusFilter) {
		return nil, ErrInvalidSeatStatus
	}
	if categoryID != nil {
		if err := s.validateCategoryForEvent(ctx, s.pool, eventID, *categoryID); err != nil {
			return nil, err
		}
	}

	list, err := s.seats.ListByEventID(ctx, s.pool, eventID, statusFilter, categoryID)
	if err != nil {
		return nil, fmt.Errorf("list seats: %w", err)
	}
	return list, nil
}

func (s *SeatService) Get(ctx context.Context, actorID, seatID uuid.UUID) (*model.Seat, error) {
	seat, err := s.seats.GetByID(ctx, s.pool, seatID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrSeatNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get seat: %w", err)
	}

	event, err := s.events.GetByID(ctx, s.pool, seat.EventID)
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

	return seat, nil
}

func (s *SeatService) Update(ctx context.Context, actorID, seatID uuid.UUID, in UpdateSeatInput) (*model.Seat, error) {
	existing, err := s.seats.GetByID(ctx, s.pool, seatID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrSeatNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get seat: %w", err)
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

	if err := ensureSeatingEditable(event.SeatingPhase); err != nil {
		return nil, err
	}

	if in.Code == "" {
		return nil, fmt.Errorf("code required")
	}
	if in.CategoryID == uuid.Nil {
		return nil, fmt.Errorf("category_id required")
	}
	if in.Status == "" {
		return nil, fmt.Errorf("status required")
	}
	if !isValidSeatStatus(in.Status) {
		return nil, ErrInvalidSeatStatus
	}

	if err := s.validateCategoryForEvent(ctx, s.pool, existing.EventID, in.CategoryID); err != nil {
		return nil, err
	}

	existing.Code = in.Code
	existing.CategoryID = in.CategoryID
	existing.Section = in.Section
	existing.Row = in.Row
	existing.Col = in.Col
	existing.Description = in.Description
	existing.Status = in.Status

	updated, err := s.seats.Update(ctx, s.pool, existing)
	if err != nil {
		return nil, fmt.Errorf("update seat: %w", err)
	}
	return updated, nil
}
