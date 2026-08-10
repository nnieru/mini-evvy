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

var (
	ErrNoOpenDraft       = errors.New("no open seating draft")
	ErrDraftItemNotFound = errors.New("seating draft item not found")
)

type seatingDraftStore interface {
	Create(ctx context.Context, db repository.DBTX, draft *model.SeatingDraft) (*model.SeatingDraft, error)
	GetOpenByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) (*model.SeatingDraft, error)
	HasApprovedByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) (bool, error)
	UpdateStatus(ctx context.Context, db repository.DBTX, draftID uuid.UUID, status model.SeatingDraftStatus) error
	GetItemByID(ctx context.Context, db repository.DBTX, itemID uuid.UUID) (*model.SeatingDraftItem, error)
	UpdateItemSeat(ctx context.Context, db repository.DBTX, itemID, seatID, categoryID uuid.UUID) error
	ListItemsByDraftID(ctx context.Context, db repository.DBTX, draftID uuid.UUID) ([]model.SeatingDraftItem, error)
	CountPreviewFiltered(ctx context.Context, db repository.DBTX, eventID uuid.UUID, q string) (int, error)
	ListPreviewPaged(ctx context.Context, db repository.DBTX, eventID uuid.UUID, pq repository.PageQuery) ([]model.SeatingPreviewRow, error)
	ListPreviewFiltered(ctx context.Context, db repository.DBTX, eventID uuid.UUID, q string) ([]model.SeatingPreviewRow, error)
	DeleteItemsByDraftID(ctx context.Context, db repository.DBTX, draftID uuid.UUID) error
	CountItemsByGuestID(ctx context.Context, db repository.DBTX, draftID, guestID uuid.UUID) (int, error)
}

type seatingDraftSeatStore interface {
	seatLookup
	ListByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID, status *model.SeatStatus, categoryID *uuid.UUID) ([]model.Seat, error)
	ClaimHeldFromAvailable(ctx context.Context, db repository.DBTX, seatID uuid.UUID) error
	ReleaseHeld(ctx context.Context, db repository.DBTX, seatID uuid.UUID) error
	TransitionHeldTo(ctx context.Context, db repository.DBTX, seatID uuid.UUID, status model.SeatStatus) error
}

type eventPhaseStore interface {
	eventLookup
	UpdateSeatingPhase(ctx context.Context, db repository.DBTX, eventID uuid.UUID, phase model.SeatingPhase) error
}

type finalizeGuestStore interface {
	guestLookup
	ListUnbookedByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]model.Guest, error)
	SoftDeleteUnbookedByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) error
}

type FinalizeService struct {
	pool        *pgxpool.Pool
	jobs        jobStore
	events      eventPhaseStore
	drafts      seatingDraftStore
	bookings    bookingStore
	guests      finalizeGuestStore
	seats       seatingDraftSeatStore
	memberships membershipChecker
	jobEnqueue  *JobService
}

func NewFinalizeService(
	pool *pgxpool.Pool,
	jobs jobStore,
	events eventPhaseStore,
	drafts seatingDraftStore,
	bookings bookingStore,
	guests finalizeGuestStore,
	seats seatingDraftSeatStore,
	memberships membershipChecker,
	jobEnqueue *JobService,
) *FinalizeService {
	return &FinalizeService{
		pool:        pool,
		jobs:        jobs,
		events:      events,
		drafts:      drafts,
		bookings:    bookings,
		guests:      guests,
		seats:       seats,
		memberships: memberships,
		jobEnqueue:  jobEnqueue,
	}
}

func (s *FinalizeService) ensureOwnerAdmin(ctx context.Context, actorID, eventID uuid.UUID) (*model.Event, error) {
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

	return event, nil
}

func (s *FinalizeService) RequestFinalize(ctx context.Context, actorID, eventID uuid.UUID) (*model.Job, error) {
	event, err := s.ensureOwnerAdmin(ctx, actorID, eventID)
	if err != nil {
		return nil, err
	}

	_, openErr := s.drafts.GetOpenByEventID(ctx, s.pool, eventID)
	hasOpenDraft := !errors.Is(openErr, repository.ErrNotFound)
	if openErr != nil && !errors.Is(openErr, repository.ErrNotFound) {
		return nil, fmt.Errorf("get open draft: %w", openErr)
	}

	switch event.SeatingPhase {
	case model.SeatingOpen, model.SeatingApproved:
	case model.SeatingPreview:
		if !hasOpenDraft {
			healed, err := s.healPhaseIfDraftMissing(ctx, eventID, event.SeatingPhase)
			if err != nil {
				return nil, err
			}
			event.SeatingPhase = healed
		}
	default:
		return nil, ErrSeatingNotOpen
	}

	exists, err := s.jobs.ExistsFinalizeInProgress(ctx, s.pool, eventID)
	if err != nil {
		return nil, fmt.Errorf("check finalize in progress: %w", err)
	}
	if exists {
		return nil, ErrFinalizeInProgress
	}

	readiness, err := s.computeSeatingReadiness(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if readiness.SlotsNeeded > 0 && !readiness.CanAssignAny {
		return nil, ErrSeatingCapacityExhausted
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

func (s *FinalizeService) healPhaseIfDraftMissing(ctx context.Context, eventID uuid.UUID, phase model.SeatingPhase) (model.SeatingPhase, error) {
	if phase != model.SeatingPreview {
		return phase, nil
	}
	_, err := s.drafts.GetOpenByEventID(ctx, s.pool, eventID)
	if err == nil {
		return phase, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return phase, fmt.Errorf("get open draft: %w", err)
	}

	hasApproved, err := s.drafts.HasApprovedByEventID(ctx, s.pool, eventID)
	if err != nil {
		return phase, fmt.Errorf("check approved drafts: %w", err)
	}
	nextPhase := model.SeatingOpen
	if hasApproved {
		nextPhase = model.SeatingApproved
	}
	if err := s.events.UpdateSeatingPhase(ctx, s.pool, eventID, nextPhase); err != nil {
		return phase, fmt.Errorf("heal seating phase: %w", err)
	}
	return nextPhase, nil
}

func (s *FinalizeService) ensureSeatingPreviewAccess(ctx context.Context, actorID, eventID uuid.UUID) error {
	event, err := s.ensureOwnerAdmin(ctx, actorID, eventID)
	if err != nil {
		return err
	}

	_, err = s.drafts.GetOpenByEventID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		if _, healErr := s.healPhaseIfDraftMissing(ctx, eventID, event.SeatingPhase); healErr != nil {
			return healErr
		}
		return ErrNoOpenDraft
	}
	if err != nil {
		return fmt.Errorf("get open draft: %w", err)
	}

	return nil
}

func (s *FinalizeService) GetSeatingPreview(ctx context.Context, actorID, eventID uuid.UUID, page, pageSize int, q string) (*PagedResult[model.SeatingPreviewRow], error) {
	if err := s.ensureSeatingPreviewAccess(ctx, actorID, eventID); err != nil {
		return nil, err
	}

	pq := repository.PageQuery{Page: page, PageSize: pageSize, Q: q}
	total, err := s.drafts.CountPreviewFiltered(ctx, s.pool, eventID, q)
	if err != nil {
		return nil, fmt.Errorf("count seating preview: %w", err)
	}

	rows, err := s.drafts.ListPreviewPaged(ctx, s.pool, eventID, pq)
	if err != nil {
		return nil, fmt.Errorf("list seating preview: %w", err)
	}

	return &PagedResult[model.SeatingPreviewRow]{
		Items:    rows,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *FinalizeService) ExportSeatingPreview(ctx context.Context, actorID, eventID uuid.UUID, q string) ([]byte, error) {
	if err := s.ensureSeatingPreviewAccess(ctx, actorID, eventID); err != nil {
		return nil, err
	}

	rows, err := s.drafts.ListPreviewFiltered(ctx, s.pool, eventID, q)
	if err != nil {
		return nil, fmt.Errorf("list seating preview: %w", err)
	}

	return buildSeatingPreviewXLSX(rows)
}

func (s *FinalizeService) GetOpenDraft(ctx context.Context, actorID, eventID uuid.UUID) (*model.SeatingDraft, error) {
	if _, err := s.ensureOwnerAdmin(ctx, actorID, eventID); err != nil {
		return nil, err
	}

	draft, err := s.drafts.GetOpenByEventID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNoOpenDraft
	}
	if err != nil {
		return nil, fmt.Errorf("get open draft: %w", err)
	}
	return draft, nil
}

func (s *FinalizeService) ReassignDraftItem(ctx context.Context, actorID, eventID, itemID, newSeatID uuid.UUID) error {
	if _, err := s.ensureOwnerAdmin(ctx, actorID, eventID); err != nil {
		return err
	}

	draft, err := s.drafts.GetOpenByEventID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNoOpenDraft
	}
	if err != nil {
		return fmt.Errorf("get open draft: %w", err)
	}

	item, err := s.drafts.GetItemByID(ctx, s.pool, itemID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrDraftItemNotFound
	}
	if err != nil {
		return fmt.Errorf("get draft item: %w", err)
	}
	if item.DraftID != draft.ID {
		return ErrDraftItemNotFound
	}

	newSeat, err := s.seats.GetByID(ctx, s.pool, newSeatID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrSeatNotFound
	}
	if err != nil {
		return fmt.Errorf("get seat: %w", err)
	}
	if newSeat.EventID != eventID {
		return ErrSeatNotFound
	}
	if newSeat.CategoryID != item.CategoryID {
		return ErrCategoryMismatch
	}
	if newSeat.ID == item.SeatID {
		return nil
	}
	if newSeat.Status != model.SeatAvailable {
		return ErrSeatNotAvailable
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.seats.ReleaseHeld(ctx, tx, item.SeatID); err != nil {
		return fmt.Errorf("release old seat: %w", err)
	}
	if err := s.seats.ClaimHeldFromAvailable(ctx, tx, newSeatID); err != nil {
		return fmt.Errorf("claim new seat: %w", err)
	}
	if err := s.drafts.UpdateItemSeat(ctx, tx, itemID, newSeatID, newSeat.CategoryID); err != nil {
		return fmt.Errorf("update draft item: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (s *FinalizeService) ApproveSeating(ctx context.Context, actorID, eventID uuid.UUID) error {
	event, err := s.ensureOwnerAdmin(ctx, actorID, eventID)
	if err != nil {
		return err
	}

	if event.SeatingPhase != model.SeatingPreview {
		return ErrSeatingNotPreview
	}

	draft, err := s.drafts.GetOpenByEventID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNoOpenDraft
	}
	if err != nil {
		return fmt.Errorf("get open draft: %w", err)
	}

	items, err := s.drafts.ListItemsByDraftID(ctx, s.pool, draft.ID)
	if err != nil {
		return fmt.Errorf("list draft items: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var createdBookings []model.SeatBooking

	for _, item := range items {
		guest, err := s.guests.GetByID(ctx, tx, item.GuestID)
		if errors.Is(err, repository.ErrNotFound) {
			return ErrGuestNotFound
		}
		if err != nil {
			return fmt.Errorf("get guest: %w", err)
		}

		seatStatus := model.SeatReserved
		booking := &model.SeatBooking{
			GuestID:    item.GuestID,
			EventID:    eventID,
			CategoryID: item.CategoryID,
			SeatID:     item.SeatID,
			Source:     model.SourceInvited,
			Status:     model.BookingPending,
			CreatedBy:  actorID,
		}
		if guest.PaidDate != nil {
			booking.Status = model.BookingPaid
			booking.PaidAt = guest.PaidDate
			seatStatus = model.SeatOccupied
		}

		if err := s.seats.TransitionHeldTo(ctx, tx, item.SeatID, seatStatus); err != nil {
			return fmt.Errorf("transition seat %s: %w", item.SeatID, err)
		}

		created, err := createBookingWithBarcode(ctx, tx, s.bookings, booking)
		if err != nil {
			return fmt.Errorf("create booking: %w", err)
		}
		createdBookings = append(createdBookings, *created)
	}

	if err := s.drafts.UpdateStatus(ctx, tx, draft.ID, model.SeatingDraftApproved); err != nil {
		return fmt.Errorf("approve draft: %w", err)
	}
	if err := s.events.UpdateSeatingPhase(ctx, tx, eventID, model.SeatingApproved); err != nil {
		return fmt.Errorf("set seating approved: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	for _, booking := range createdBookings {
		_, err := s.jobEnqueue.Enqueue(ctx, jobtype.SendInvitation, jobtype.SendInvitationPayload{
			BookingID: booking.ID,
			GuestID:   booking.GuestID,
			EventID:   eventID,
		})
		if err != nil {
			return fmt.Errorf("enqueue invitation: %w", err)
		}
	}

	return nil
}

func (s *FinalizeService) RejectSeating(ctx context.Context, actorID, eventID uuid.UUID) error {
	if _, err := s.ensureOwnerAdmin(ctx, actorID, eventID); err != nil {
		return err
	}

	draft, err := s.drafts.GetOpenByEventID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNoOpenDraft
	}
	if err != nil {
		return fmt.Errorf("get open draft: %w", err)
	}

	items, err := s.drafts.ListItemsByDraftID(ctx, s.pool, draft.ID)
	if err != nil {
		return fmt.Errorf("list draft items: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		if err := s.seats.ReleaseHeld(ctx, tx, item.SeatID); err != nil {
			if !errors.Is(err, repository.ErrSeatNotAvailable) {
				return fmt.Errorf("release seat %s: %w", item.SeatID, err)
			}
		}
	}

	hasApproved, err := s.drafts.HasApprovedByEventID(ctx, tx, eventID)
	if err != nil {
		return fmt.Errorf("check approved drafts: %w", err)
	}

	if err := s.guests.SoftDeleteUnbookedByEventID(ctx, tx, eventID); err != nil {
		return fmt.Errorf("soft delete unbooked guests: %w", err)
	}

	if err := s.drafts.DeleteItemsByDraftID(ctx, tx, draft.ID); err != nil {
		return fmt.Errorf("delete draft items: %w", err)
	}

	if err := s.drafts.UpdateStatus(ctx, tx, draft.ID, model.SeatingDraftRejected); err != nil {
		return fmt.Errorf("reject draft: %w", err)
	}

	nextPhase := model.SeatingOpen
	if hasApproved {
		nextPhase = model.SeatingApproved
	}
	if err := s.events.UpdateSeatingPhase(ctx, tx, eventID, nextPhase); err != nil {
		return fmt.Errorf("set seating phase: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// RevertSeatingToOpen resets phase after a failed finalize job (best-effort).
func (s *FinalizeService) RevertSeatingToOpen(ctx context.Context, eventID uuid.UUID) {
	hasApproved, err := s.drafts.HasApprovedByEventID(ctx, s.pool, eventID)
	if err != nil {
		_ = err
		hasApproved = false
	}
	nextPhase := model.SeatingOpen
	if hasApproved {
		nextPhase = model.SeatingApproved
	}
	if err := s.events.UpdateSeatingPhase(ctx, s.pool, eventID, nextPhase); err != nil {
		_ = err
	}
}
