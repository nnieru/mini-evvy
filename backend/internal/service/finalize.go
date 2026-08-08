package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/jobtype"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
)

type eventPhaseStore interface {
	eventLookup
	UpdateSeatingPhase(ctx context.Context, db repository.DBTX, eventID uuid.UUID, phase model.SeatingPhase) error
}

type seatingPreviewStore interface {
	ListSeatingPreviewByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]model.SeatingPreviewRow, error)
	ListActiveByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]model.SeatBooking, error)
	Update(ctx context.Context, db repository.DBTX, b *model.SeatBooking) (*model.SeatBooking, error)
}

type FinalizeService struct {
	pool        *pgxpool.Pool
	jobs        jobStore
	events      eventPhaseStore
	bookings    seatingPreviewStore
	seats       seatLookup
	memberships membershipChecker
	jobEnqueue  *JobService
}

func NewFinalizeService(
	pool *pgxpool.Pool,
	jobs jobStore,
	events eventPhaseStore,
	bookings seatingPreviewStore,
	seats seatLookup,
	memberships membershipChecker,
	jobEnqueue *JobService,
) *FinalizeService {
	return &FinalizeService{
		pool:        pool,
		jobs:        jobs,
		events:      events,
		bookings:    bookings,
		seats:       seats,
		memberships: memberships,
		jobEnqueue:  jobEnqueue,
	}
}

func (s *FinalizeService) RequestFinalize(ctx context.Context, actorID, eventID uuid.UUID) (*model.Job, error) {
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

	if event.SeatingPhase != model.SeatingOpen {
		return nil, ErrSeatingNotOpen
	}

	exists, err := s.jobs.ExistsFinalizeInProgress(ctx, s.pool, eventID)
	if err != nil {
		return nil, fmt.Errorf("check finalize in progress: %w", err)
	}
	if exists {
		return nil, ErrFinalizeInProgress
	}

	job, err := s.jobEnqueue.Enqueue(ctx, jobtype.FinalizeSeating, jobtype.FinalizeSeatingPayload{
		EventID: eventID,
		ActorID: actorID,
	})
	if err != nil {
		return nil, fmt.Errorf("enqueue finalize job: %w", err)
	}

	return job, nil
}

func (s *FinalizeService) GetSeatingPreview(ctx context.Context, actorID, eventID uuid.UUID) ([]model.SeatingPreviewRow, error) {
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

	if event.SeatingPhase != model.SeatingPreview && event.SeatingPhase != model.SeatingApproved {
		return nil, ErrSeatingNotPreview
	}

	rows, err := s.bookings.ListSeatingPreviewByEventID(ctx, s.pool, eventID)
	if err != nil {
		return nil, fmt.Errorf("list seating preview: %w", err)
	}
	return rows, nil
}

func (s *FinalizeService) ApproveSeating(ctx context.Context, actorID, eventID uuid.UUID) error {
	event, err := s.events.GetByID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrEventNotFound
	}
	if err != nil {
		return fmt.Errorf("get event: %w", err)
	}

	can, err := s.memberships.HasRole(ctx, s.pool, actorID, event.OrganizationID, model.RoleOwner, model.RoleAdmin)
	if err != nil {
		return fmt.Errorf("check role: %w", err)
	}
	if !can {
		return ErrForbidden
	}

	if event.SeatingPhase != model.SeatingPreview {
		return ErrSeatingNotPreview
	}

	bookings, err := s.bookings.ListActiveByEventID(ctx, s.pool, eventID)
	if err != nil {
		return fmt.Errorf("list active bookings: %w", err)
	}

	for _, booking := range bookings {
		_, err := s.jobEnqueue.Enqueue(ctx, jobtype.SendInvitation, jobtype.SendInvitationPayload{
			BookingID: booking.ID,
			GuestID:   booking.GuestID,
			EventID:   eventID,
		})
		if err != nil {
			return fmt.Errorf("enqueue invitation: %w", err)
		}
	}

	if err := s.events.UpdateSeatingPhase(ctx, s.pool, eventID, model.SeatingApproved); err != nil {
		return fmt.Errorf("set seating approved: %w", err)
	}

	return nil
}

func (s *FinalizeService) RejectSeating(ctx context.Context, actorID, eventID uuid.UUID) error {
	event, err := s.events.GetByID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrEventNotFound
	}
	if err != nil {
		return fmt.Errorf("get event: %w", err)
	}

	can, err := s.memberships.HasRole(ctx, s.pool, actorID, event.OrganizationID, model.RoleOwner, model.RoleAdmin)
	if err != nil {
		return fmt.Errorf("check role: %w", err)
	}
	if !can {
		return ErrForbidden
	}

	if event.SeatingPhase != model.SeatingPreview && event.SeatingPhase != model.SeatingApproved {
		return ErrSeatingNotPreview
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	bookings, err := s.bookings.ListActiveByEventID(ctx, tx, eventID)
	if err != nil {
		return fmt.Errorf("list active bookings: %w", err)
	}

	for _, booking := range bookings {
		if booking.Source != model.SourceInvited {
			continue
		}
		if booking.Status != model.BookingPending && booking.Status != model.BookingPaid {
			continue
		}

		booking.Status = model.BookingCancelled
		updatedBy := actorID
		booking.UpdatedBy = &updatedBy

		if _, err := s.bookings.Update(ctx, tx, &booking); err != nil {
			return fmt.Errorf("cancel booking: %w", err)
		}
		if err := s.seats.UpdateStatus(ctx, tx, booking.SeatID, model.SeatAvailable); err != nil {
			return fmt.Errorf("free seat: %w", err)
		}
	}

	if err := s.events.UpdateSeatingPhase(ctx, tx, eventID, model.SeatingOpen); err != nil {
		return fmt.Errorf("set seating open: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// RevertSeatingToOpen resets phase after a failed finalize job (best-effort).
func (s *FinalizeService) RevertSeatingToOpen(ctx context.Context, eventID uuid.UUID) {
	if err := s.events.UpdateSeatingPhase(ctx, s.pool, eventID, model.SeatingOpen); err != nil {
		// best-effort; worker logs separately
		_ = err
	}
}
