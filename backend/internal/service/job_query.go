package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/jobtype"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
)

var (
	ErrJobNotFound         = errors.New("job not found")
	ErrGuestEmailMissing   = errors.New("guest email is missing")
	ErrBookingNotResendable = errors.New("booking cannot receive invitation")
)

type jobQueryStore interface {
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.Job, error)
	ListByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]model.Job, error)
}

type jobEnqueuer interface {
	Enqueue(ctx context.Context, jobType string, data any) (*model.Job, error)
}

type resendBookingLookup interface {
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.SeatBooking, error)
}

type JobQueryService struct {
	pool        *pgxpool.Pool
	jobs        jobQueryStore
	bookings    resendBookingLookup
	guests      guestLookup
	events      eventLookup
	memberships membershipChecker
	enqueue     jobEnqueuer
}

func NewJobQueryService(
	pool *pgxpool.Pool,
	jobs jobQueryStore,
	bookings resendBookingLookup,
	guests guestLookup,
	events eventLookup,
	memberships membershipChecker,
	enqueue jobEnqueuer,
) *JobQueryService {
	return &JobQueryService{
		pool:        pool,
		jobs:        jobs,
		bookings:    bookings,
		guests:      guests,
		events:      events,
		memberships: memberships,
		enqueue:     enqueue,
	}
}

type jobEventPayload struct {
	EventID string `json:"event_id"`
}

func eventIDFromJobData(data json.RawMessage) (uuid.UUID, error) {
	var payload jobEventPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(payload.EventID)
}

func (s *JobQueryService) ListByEvent(ctx context.Context, actorID, eventID uuid.UUID) ([]model.Job, error) {
	event, err := s.events.GetByID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	ok, err := s.memberships.HasRole(ctx, s.pool, actorID, event.OrganizationID, model.RoleOwner, model.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if !ok {
		return nil, ErrForbidden
	}

	list, err := s.jobs.ListByEventID(ctx, s.pool, eventID)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	return list, nil
}

func (s *JobQueryService) Get(ctx context.Context, actorID, jobID uuid.UUID) (*model.Job, error) {
	job, err := s.jobs.GetByID(ctx, s.pool, jobID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}

	eventID, err := eventIDFromJobData(job.Data)
	if err != nil {
		return nil, fmt.Errorf("parse job event id: %w", err)
	}

	event, err := s.events.GetByID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	ok, err := s.memberships.HasRole(ctx, s.pool, actorID, event.OrganizationID, model.RoleOwner, model.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if !ok {
		return nil, ErrForbidden
	}

	return job, nil
}

func (s *JobQueryService) ResendInvitation(ctx context.Context, actorID, bookingID uuid.UUID) (*model.Job, error) {
	booking, err := s.bookings.GetByID(ctx, s.pool, bookingID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrBookingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get booking: %w", err)
	}

	if booking.Status == model.BookingCancelled {
		return nil, ErrBookingNotResendable
	}

	event, err := s.events.GetByID(ctx, s.pool, booking.EventID)
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

	guest, err := s.guests.GetByID(ctx, s.pool, booking.GuestID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrGuestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get guest: %w", err)
	}

	if strings.TrimSpace(guest.Email) == "" {
		return nil, ErrGuestEmailMissing
	}

	job, err := s.enqueue.Enqueue(ctx, jobtype.SendInvitation, jobtype.SendInvitationPayload{
		BookingID: booking.ID,
		GuestID:   booking.GuestID,
		EventID:   booking.EventID,
	})
	if err != nil {
		return nil, fmt.Errorf("enqueue send invitation: %w", err)
	}

	return job, nil
}
