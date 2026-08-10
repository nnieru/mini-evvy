package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
)

var (
	ErrBookingNotFound         = errors.New("booking not found")
	ErrSeatNotAvailable        = errors.New("seat not available")
	ErrInvalidBookingStatus    = errors.New("invalid booking status")
	ErrInvalidBookingSource    = errors.New("invalid booking source")
	ErrInvalidBookingRequest   = errors.New("invalid booking request")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrCategoryMismatch        = errors.New("guest and seat category mismatch")
	ErrSeatsDifferentCategory  = errors.New("all seats must be in the same category")
)

type bookingStore interface {
	Create(ctx context.Context, db repository.DBTX, b *model.SeatBooking) (*model.SeatBooking, error)
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.SeatBooking, error)
	ListByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]model.SeatBooking, error)
	ListActiveByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]model.SeatBooking, error)
	ListByEventIDPaged(ctx context.Context, db repository.DBTX, eventID uuid.UUID, q repository.BookingListQuery) ([]model.BookingListRow, error)
	CountByEventIDFiltered(ctx context.Context, db repository.DBTX, eventID uuid.UUID, q repository.BookingListQuery) (int, error)
	ListExpiredUnpaid(ctx context.Context, db repository.DBTX, olderThan time.Time) ([]model.SeatBooking, error)
	Update(ctx context.Context, db repository.DBTX, b *model.SeatBooking) (*model.SeatBooking, error)
	SoftDelete(ctx context.Context, db repository.DBTX, b *model.SeatBooking) (*model.SeatBooking, error)
	CountActiveByGuestID(ctx context.Context, db repository.DBTX, guestID uuid.UUID) (int, error)
	SetInvitationEmailPending(ctx context.Context, db repository.DBTX, id uuid.UUID) error
	UpdateInvitationEmailResult(
		ctx context.Context,
		db repository.DBTX,
		id uuid.UUID,
		status model.InvitationEmailStatus,
		sentAt *time.Time,
	) error
	ReconcileInvitationEmailStatusByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) error
}

type guestLookup interface {
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.Guest, error)
}

type guestCreator interface {
	guestLookup
	Create(ctx context.Context, db repository.DBTX, g *model.Guest) (*model.Guest, error)
	Update(ctx context.Context, db repository.DBTX, g *model.Guest) (*model.Guest, error)
	GetByEventNameEmailCategory(ctx context.Context, db repository.DBTX, eventID, categoryID uuid.UUID, name, email string) (*model.Guest, error)
}

type bookingCategoryLookup interface {
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.SeatCategory, error)
}

type BookingService struct {
	pool        *pgxpool.Pool
	bookings    bookingStore
	events      eventLookup
	guests      guestCreator
	seats       seatLookup
	categories  bookingCategoryLookup
	memberships membershipChecker
	jobEnqueue  *JobService
}

func NewBookingService(
	pool *pgxpool.Pool,
	bookings bookingStore,
	events eventLookup,
	guests guestCreator,
	seats seatLookup,
	categories bookingCategoryLookup,
	memberships membershipChecker,
	jobEnqueue *JobService,
) *BookingService {
	return &BookingService{
		pool:        pool,
		bookings:    bookings,
		events:      events,
		guests:      guests,
		seats:       seats,
		categories:  categories,
		memberships: memberships,
		jobEnqueue:  jobEnqueue,
	}
}

type seatLookup interface {
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.Seat, error)
	UpdateStatus(ctx context.Context, db repository.DBTX, seatID uuid.UUID, status model.SeatStatus) error
	ClaimFromAvailable(ctx context.Context, db repository.DBTX, seatID uuid.UUID, status model.SeatStatus) error
}

type CreateBookingInput struct {
	GuestID    *uuid.UUID
	SeatID     uuid.UUID
	Source     model.BookingSource
	Notes      *string
	Name       string
	Email      string
	CategoryID uuid.UUID
}

type UpdateBookingInput struct {
	Status model.BookingStatus
	Notes  *string
}

type CreateBookingBatchItemInput struct {
	SeatID uuid.UUID
	Name   string
	Email  string
}

type CreateBookingBatchInput struct {
	Notes *string
	Items []CreateBookingBatchItemInput
}

func isValidBookingSource(source model.BookingSource) bool {
	return source == model.SourceInvited || source == model.SourceOnsite
}

func isValidStatusTransition(from, to model.BookingStatus) bool {
	switch from {
	case model.BookingPending:
		return to == model.BookingNotPaid || to == model.BookingPaid || to == model.BookingCancelled
	case model.BookingNotPaid:
		return to == model.BookingPaid || to == model.BookingCancelled
	case model.BookingPaid:
		return to == model.BookingNotPaid || to == model.BookingCancelled
	default:
		return false
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func claimSeatFromAvailable(ctx context.Context, db repository.DBTX, seats seatLookup, seatID uuid.UUID, status model.SeatStatus) error {
	err := seats.ClaimFromAvailable(ctx, db, seatID, status)
	if errors.Is(err, repository.ErrSeatNotAvailable) {
		return ErrSeatNotAvailable
	}
	return err
}

func (s *BookingService) validateCategoryForEvent(ctx context.Context, db repository.DBTX, eventID, categoryID uuid.UUID) error {
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

func (s *BookingService) Create(ctx context.Context, actorID, eventID uuid.UUID, in CreateBookingInput) (*model.SeatBooking, error) {
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

	seat, err := s.seats.GetByID(ctx, s.pool, in.SeatID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrSeatNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get seat: %w", err)
	}
	if seat.EventID != eventID {
		return nil, ErrSeatNotFound
	}
	if seat.Status != model.SeatAvailable {
		return nil, ErrSeatNotAvailable
	}

	hasGuestID := in.GuestID != nil && *in.GuestID != uuid.Nil
	hasInlineGuest := in.Name != "" || in.Email != "" || in.CategoryID != uuid.Nil
	if hasGuestID && hasInlineGuest {
		return nil, ErrInvalidBookingRequest
	}
	if !hasGuestID && !hasInlineGuest {
		return nil, ErrInvalidBookingRequest
	}

	source := in.Source
	var guest *model.Guest

	if hasGuestID {
		guest, err = s.guests.GetByID(ctx, s.pool, *in.GuestID)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrGuestNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("get guest: %w", err)
		}
		if guest.EventID != eventID {
			return nil, ErrGuestNotFound
		}
		if guest.CategoryID != seat.CategoryID {
			return nil, ErrCategoryMismatch
		}
	} else {
		if in.Name == "" || in.Email == "" || in.CategoryID == uuid.Nil {
			return nil, ErrInvalidBookingRequest
		}
		if in.CategoryID != seat.CategoryID {
			return nil, ErrCategoryMismatch
		}
		if err := s.validateCategoryForEvent(ctx, s.pool, eventID, in.CategoryID); err != nil {
			return nil, err
		}
		source = model.SourceInvited
	}

	if source == "" {
		source = model.SourceInvited
	}
	if !isValidBookingSource(source) {
		return nil, ErrInvalidBookingSource
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if hasInlineGuest {
		guest, err = s.findOrCreateGuest(ctx, tx, eventID, seat.CategoryID, in.Name, in.Email, 1)
		if err != nil {
			return nil, err
		}
	}

	if err := claimSeatFromAvailable(ctx, tx, s.seats, in.SeatID, model.SeatReserved); err != nil {
		return nil, err
	}

	booking := &model.SeatBooking{
		GuestID:    guest.ID,
		EventID:    eventID,
		CategoryID: seat.CategoryID,
		SeatID:     in.SeatID,
		Source:     source,
		Status:     model.BookingPending,
		Notes:      in.Notes,
		CreatedBy:  actorID,
	}

	created, err := createBookingWithBarcode(ctx, tx, s.bookings, booking)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSeatNotAvailable
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}

func (s *BookingService) CreateBatch(ctx context.Context, actorID, eventID uuid.UUID, in CreateBookingBatchInput) ([]model.SeatBooking, error) {
	if len(in.Items) == 0 {
		return nil, ErrInvalidBookingRequest
	}

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

	seenSeats := make(map[uuid.UUID]struct{}, len(in.Items))
	type resolvedItem struct {
		seat model.Seat
		name string
		email string
	}
	resolved := make([]resolvedItem, 0, len(in.Items))

	var categoryID uuid.UUID
	var categorySet bool

	for _, item := range in.Items {
		if item.SeatID == uuid.Nil {
			return nil, ErrInvalidBookingRequest
		}
		if item.Name == "" || item.Email == "" {
			return nil, ErrInvalidBookingRequest
		}
		if _, dup := seenSeats[item.SeatID]; dup {
			return nil, ErrInvalidBookingRequest
		}
		seenSeats[item.SeatID] = struct{}{}

		seat, err := s.seats.GetByID(ctx, s.pool, item.SeatID)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSeatNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("get seat: %w", err)
		}
		if seat.EventID != eventID {
			return nil, ErrSeatNotFound
		}
		if seat.Status != model.SeatAvailable {
			return nil, ErrSeatNotAvailable
		}

		if !categorySet {
			categoryID = seat.CategoryID
			categorySet = true
		} else if seat.CategoryID != categoryID {
			return nil, ErrSeatsDifferentCategory
		}

		if err := s.validateCategoryForEvent(ctx, s.pool, eventID, seat.CategoryID); err != nil {
			return nil, err
		}

		resolved = append(resolved, resolvedItem{
			seat:  *seat,
			name:  item.Name,
			email: item.Email,
		})
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	created := make([]model.SeatBooking, 0, len(resolved))
	for _, item := range resolved {
		guest, err := s.findOrCreateGuest(ctx, tx, eventID, item.seat.CategoryID, item.name, item.email, 1)
		if err != nil {
			return nil, err
		}

		if err := claimSeatFromAvailable(ctx, tx, s.seats, item.seat.ID, model.SeatReserved); err != nil {
			return nil, err
		}

		booking := &model.SeatBooking{
			GuestID:    guest.ID,
			EventID:    eventID,
			CategoryID: item.seat.CategoryID,
			SeatID:     item.seat.ID,
			Source:     model.SourceInvited,
			Status:     model.BookingPending,
			Notes:      in.Notes,
			CreatedBy:  actorID,
		}

		row, err := createBookingWithBarcode(ctx, tx, s.bookings, booking)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, ErrSeatNotAvailable
			}
			return nil, err
		}

		created = append(created, *row)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}

func (s *BookingService) ListByEvent(ctx context.Context, actorID, eventID uuid.UUID) ([]model.SeatBooking, error) {
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

	list, err := s.bookings.ListByEventID(ctx, s.pool, eventID)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	return list, nil
}

type ListBookingsFilter struct {
	PaymentStatus string
	Q             string
	Page          int
	PageSize      int
}

type ListBookingsResult struct {
	Items    []model.BookingListRow
	Total    int
	Page     int
	PageSize int
}

func normalizeListBookingsFilter(f ListBookingsFilter) ListBookingsFilter {
	switch f.PaymentStatus {
	case "paid", "unpaid":
	default:
		f.PaymentStatus = "all"
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	return f
}

func (s *BookingService) ListByEventPaged(ctx context.Context, actorID, eventID uuid.UUID, filter ListBookingsFilter) (*ListBookingsResult, error) {
	filter = normalizeListBookingsFilter(filter)

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

	if err := s.bookings.ReconcileInvitationEmailStatusByEventID(ctx, s.pool, eventID); err != nil {
		return nil, fmt.Errorf("reconcile invitation status: %w", err)
	}

	total, err := s.bookings.CountByEventIDFiltered(ctx, s.pool, eventID, repository.BookingListQuery{
		PaymentStatus: filter.PaymentStatus,
		Q:             filter.Q,
	})
	if err != nil {
		return nil, fmt.Errorf("count bookings: %w", err)
	}

	items, err := s.bookings.ListByEventIDPaged(ctx, s.pool, eventID, repository.BookingListQuery{
		PaymentStatus: filter.PaymentStatus,
		Q:             filter.Q,
		Page:          filter.Page,
		PageSize:      filter.PageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}

	return &ListBookingsResult{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

func (s *BookingService) Get(ctx context.Context, actorID, bookingID uuid.UUID) (*model.SeatBooking, error) {
	booking, err := s.bookings.GetByID(ctx, s.pool, bookingID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrBookingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get booking: %w", err)
	}

	event, err := s.events.GetByID(ctx, s.pool, booking.EventID)
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

	return booking, nil
}

func (s *BookingService) Update(ctx context.Context, actorID, bookingID uuid.UUID, in UpdateBookingInput) (*model.SeatBooking, error) {
	existing, err := s.bookings.GetByID(ctx, s.pool, bookingID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrBookingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get booking: %w", err)
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

	if in.Status == "" {
		return nil, ErrInvalidBookingStatus
	}
	if !isValidStatusTransition(existing.Status, in.Status) {
		return nil, ErrInvalidStatusTransition
	}

	wasPaid := existing.Status == model.BookingPaid

	if in.Notes != nil {
		existing.Notes = in.Notes
	}

	existing.Status = in.Status
	updatedBy := actorID
	existing.UpdatedBy = &updatedBy

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	switch in.Status {
	case model.BookingPaid:
		if existing.PaidAt == nil {
			now := time.Now()
			existing.PaidAt = &now
		}
		if err := s.seats.UpdateStatus(ctx, tx, existing.SeatID, model.SeatOccupied); err != nil {
			return nil, fmt.Errorf("update seat status: %w", err)
		}
	case model.BookingNotPaid:
		if wasPaid {
			existing.PaidAt = nil
			if err := s.seats.UpdateStatus(ctx, tx, existing.SeatID, model.SeatReserved); err != nil {
				return nil, fmt.Errorf("update seat status: %w", err)
			}
		}
	case model.BookingCancelled:
		if err := s.seats.UpdateStatus(ctx, tx, existing.SeatID, model.SeatAvailable); err != nil {
			return nil, fmt.Errorf("update seat status: %w", err)
		}
	}

	var updated *model.SeatBooking
	if in.Status == model.BookingPaid && !wasPaid {
		updated, err = ensureBookingBarcode(ctx, tx, s.bookings, existing)
	} else {
		updated, err = s.bookings.Update(ctx, tx, existing)
	}
	if err != nil {
		return nil, fmt.Errorf("update booking: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return updated, nil
}

func (s *BookingService) Delete(ctx context.Context, actorID, bookingID uuid.UUID) error {
	existing, err := s.bookings.GetByID(ctx, s.pool, bookingID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrBookingNotFound
	}
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}

	event, err := s.events.GetByID(ctx, s.pool, existing.EventID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrEventNotFound
	}
	if err != nil {
		return fmt.Errorf("get event: %w", err)
	}

	can, err := s.memberships.IsMember(ctx, s.pool, actorID, event.OrganizationID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !can {
		return ErrForbidden
	}

	updatedBy := actorID
	existing.UpdatedBy = &updatedBy

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if existing.Status != model.BookingCancelled {
		existing.Status = model.BookingCancelled
		if err := s.seats.UpdateStatus(ctx, tx, existing.SeatID, model.SeatAvailable); err != nil {
			return fmt.Errorf("update seat status: %w", err)
		}
	}

	if _, err := s.bookings.SoftDelete(ctx, tx, existing); err != nil {
		return fmt.Errorf("soft delete booking: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// ExpireUnpaidBookings cancels pending/not_paid bookings older than cutoff and frees reserved seats.
func (s *BookingService) ExpireUnpaidBookings(ctx context.Context, cutoff time.Time) (int, error) {
	list, err := s.bookings.ListExpiredUnpaid(ctx, s.pool, cutoff)
	if err != nil {
		return 0, fmt.Errorf("list expired unpaid: %w", err)
	}

	var expired int
	for _, booking := range list {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return expired, fmt.Errorf("begin transaction: %w", err)
		}

		current, err := s.bookings.GetByID(ctx, tx, booking.ID)
		if errors.Is(err, repository.ErrNotFound) {
			_ = tx.Rollback(ctx)
			continue
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return expired, fmt.Errorf("get booking: %w", err)
		}
		if current.Status != model.BookingPending && current.Status != model.BookingNotPaid {
			_ = tx.Rollback(ctx)
			continue
		}

		current.Status = model.BookingCancelled
		if err := s.seats.UpdateStatus(ctx, tx, current.SeatID, model.SeatAvailable); err != nil {
			_ = tx.Rollback(ctx)
			return expired, fmt.Errorf("free seat: %w", err)
		}
		if _, err := s.bookings.Update(ctx, tx, current); err != nil {
			_ = tx.Rollback(ctx)
			return expired, fmt.Errorf("cancel booking: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return expired, fmt.Errorf("commit expire: %w", err)
		}
		expired++
	}

	return expired, nil
}
