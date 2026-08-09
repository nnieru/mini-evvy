package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
)

var (
	ErrAttendanceNotFound            = errors.New("attendance not found")
	ErrPaidBookingNotFound           = errors.New("paid booking not found for guest and seat")
	ErrInvalidBarcode                = errors.New("invalid barcode")
	ErrAlreadyCheckedIn              = errors.New("guest already checked in for this seat")
	ErrInvalidAttendanceStatus       = errors.New("invalid attendance status")
	ErrInvalidAttendanceTransition   = errors.New("invalid attendance status transition")
)

type attendanceStore interface {
	Create(ctx context.Context, db repository.DBTX, a *model.AttendanceLog) (*model.AttendanceLog, error)
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.AttendanceLog, error)
	ListByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]model.AttendanceLog, error)
	CountByEventIDFiltered(ctx context.Context, db repository.DBTX, eventID uuid.UUID, q string) (int, error)
	ListByEventIDPaged(ctx context.Context, db repository.DBTX, eventID uuid.UUID, pq repository.PageQuery) ([]model.AttendanceLog, error)
	Update(ctx context.Context, db repository.DBTX, a *model.AttendanceLog) (*model.AttendanceLog, error)
	SoftDelete(ctx context.Context, db repository.DBTX, id, updatedBy uuid.UUID) error
	ExistsCheckedIn(ctx context.Context, db repository.DBTX, eventID, guestID, seatID uuid.UUID) (bool, error)
}

type attendanceBookingLookup interface {
	GetPaidByGuestSeat(ctx context.Context, db repository.DBTX, eventID, guestID, seatID uuid.UUID) (*model.SeatBooking, error)
	GetByBarcode(ctx context.Context, db repository.DBTX, barcode string) (*model.SeatBooking, error)
}

type AttendanceService struct {
	pool        *pgxpool.Pool
	attendance  attendanceStore
	guests      guestLookup
	seats       seatLookup
	bookings    attendanceBookingLookup
	events      eventLookup
	memberships membershipChecker
}

func NewAttendanceService(
	pool *pgxpool.Pool,
	attendance attendanceStore,
	guests guestLookup,
	seats seatLookup,
	bookings attendanceBookingLookup,
	events eventLookup,
	memberships membershipChecker,
) *AttendanceService {
	return &AttendanceService{
		pool:        pool,
		attendance:  attendance,
		guests:      guests,
		seats:       seats,
		bookings:    bookings,
		events:      events,
		memberships: memberships,
	}
}

type CreateAttendanceInput struct {
	Barcode string
	GuestID uuid.UUID
	SeatID  uuid.UUID
	Message *string
}

func (s *AttendanceService) Create(ctx context.Context, actorID, eventID uuid.UUID, in CreateAttendanceInput) (*model.AttendanceLog, error) {
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

	barcode := strings.TrimSpace(in.Barcode)
	hasBarcode := barcode != ""
	hasGuestSeat := in.GuestID != uuid.Nil && in.SeatID != uuid.Nil

	if hasBarcode && hasGuestSeat {
		return nil, ErrInvalidBarcode
	}
	if !hasBarcode && !hasGuestSeat {
		return nil, fmt.Errorf("guest_id and seat_id or barcode required")
	}

	var guestID, seatID uuid.UUID

	if hasBarcode {
		booking, err := s.bookings.GetByBarcode(ctx, s.pool, barcode)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrPaidBookingNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("get booking by barcode: %w", err)
		}
		if booking.EventID != eventID {
			return nil, ErrPaidBookingNotFound
		}
		if booking.Status != model.BookingPaid {
			return nil, ErrPaidBookingNotFound
		}
		guestID = booking.GuestID
		seatID = booking.SeatID
	} else {
		guest, err := s.guests.GetByID(ctx, s.pool, in.GuestID)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrGuestNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("get guest: %w", err)
		}
		if guest.EventID != eventID {
			return nil, ErrGuestNotFound
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

		_, err = s.bookings.GetPaidByGuestSeat(ctx, s.pool, eventID, in.GuestID, in.SeatID)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrPaidBookingNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("get paid booking: %w", err)
		}

		guestID = in.GuestID
		seatID = in.SeatID
	}

	exists, err := s.attendance.ExistsCheckedIn(ctx, s.pool, eventID, guestID, seatID)
	if err != nil {
		return nil, fmt.Errorf("check existing check-in: %w", err)
	}
	if exists {
		return nil, ErrAlreadyCheckedIn
	}

	log := &model.AttendanceLog{
		GuestID:   guestID,
		EventID:   eventID,
		SeatID:    seatID,
		Status:    model.AttendanceStatusCheckedIn,
		Message:   in.Message,
		CreatedBy: actorID,
	}

	created, err := s.attendance.Create(ctx, s.pool, log)
	if err != nil {
		return nil, fmt.Errorf("create attendance log: %w", err)
	}

	return created, nil
}

func (s *AttendanceService) ListByEvent(ctx context.Context, actorID, eventID uuid.UUID) ([]model.AttendanceLog, error) {
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

	list, err := s.attendance.ListByEventID(ctx, s.pool, eventID)
	if err != nil {
		return nil, fmt.Errorf("list attendance: %w", err)
	}
	return list, nil
}

func (s *AttendanceService) ListByEventPaged(ctx context.Context, actorID, eventID uuid.UUID, page, pageSize int, q string) (*PagedResult[model.AttendanceLog], error) {
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

	pq := repository.PageQuery{Page: page, PageSize: pageSize, Q: q}
	total, err := s.attendance.CountByEventIDFiltered(ctx, s.pool, eventID, q)
	if err != nil {
		return nil, fmt.Errorf("count attendance: %w", err)
	}

	list, err := s.attendance.ListByEventIDPaged(ctx, s.pool, eventID, pq)
	if err != nil {
		return nil, fmt.Errorf("list attendance: %w", err)
	}

	return &PagedResult[model.AttendanceLog]{
		Items:    list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *AttendanceService) Get(ctx context.Context, actorID, attendanceID uuid.UUID) (*model.AttendanceLog, error) {
	log, err := s.attendance.GetByID(ctx, s.pool, attendanceID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrAttendanceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get attendance: %w", err)
	}

	event, err := s.events.GetByID(ctx, s.pool, log.EventID)
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

	return log, nil
}

type UpdateAttendanceInput struct {
	Status  model.AttendanceStatus
	Message *string
}

func (s *AttendanceService) Update(ctx context.Context, actorID, attendanceID uuid.UUID, in UpdateAttendanceInput) (*model.AttendanceLog, error) {
	existing, err := s.attendance.GetByID(ctx, s.pool, attendanceID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrAttendanceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get attendance: %w", err)
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
		return nil, ErrInvalidAttendanceStatus
	}
	if in.Status != model.AttendanceStatusNotCheckedIn {
		return nil, ErrInvalidAttendanceStatus
	}
	if existing.Status != model.AttendanceStatusCheckedIn {
		return nil, ErrInvalidAttendanceTransition
	}

	existing.Status = in.Status
	if in.Message != nil {
		existing.Message = in.Message
	}
	updatedBy := actorID
	existing.UpdatedBy = &updatedBy

	updated, err := s.attendance.Update(ctx, s.pool, existing)
	if err != nil {
		return nil, fmt.Errorf("update attendance: %w", err)
	}

	return updated, nil
}

func (s *AttendanceService) Delete(ctx context.Context, actorID, attendanceID uuid.UUID) error {
	existing, err := s.attendance.GetByID(ctx, s.pool, attendanceID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrAttendanceNotFound
	}
	if err != nil {
		return fmt.Errorf("get attendance: %w", err)
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

	if err := s.attendance.SoftDelete(ctx, s.pool, attendanceID, actorID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrAttendanceNotFound
		}
		return fmt.Errorf("soft delete attendance: %w", err)
	}

	return nil
}
