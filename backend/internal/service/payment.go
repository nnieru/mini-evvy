package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
	"github.com/nnieru/mini-evvy/internal/jobtype"
)

var (
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrInvalidPaymentStatus = errors.New("invalid payment status")
	ErrInvalidPaymentAmount = errors.New("invalid payment amount")
	ErrInvalidCurrency      = errors.New("invalid currency")
	ErrBookingNotPayable    = errors.New("booking not payable")
)

var currencyRegex = regexp.MustCompile(`^[A-Z]{3}$`)

type paymentStore interface {
	Create(ctx context.Context, db repository.DBTX, p *model.Payment) (*model.Payment, error)
	ListByBookingID(ctx context.Context, db repository.DBTX, bookingID uuid.UUID) ([]model.Payment, error)
}

type bookingLookup interface {
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.SeatBooking, error)
	Update(ctx context.Context, db repository.DBTX, b *model.SeatBooking) (*model.SeatBooking, error)
}

type seatStatusUpdater interface {
	UpdateStatus(ctx context.Context, db repository.DBTX, seatID uuid.UUID, status model.SeatStatus) error
}

type guestUpdater interface {
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.Guest, error)
	Update(ctx context.Context, db repository.DBTX, g *model.Guest) (*model.Guest, error)
}

type PaymentService struct {
	pool        *pgxpool.Pool
	payments    paymentStore
	bookings    bookingLookup
	events      eventLookup
	seats       seatStatusUpdater
	guests      guestUpdater
	memberships membershipChecker
	jobEnqueue  *JobService
}

func NewPaymentService(
	pool *pgxpool.Pool,
	payments paymentStore,
	bookings bookingLookup,
	events eventLookup,
	seats seatStatusUpdater,
	guests guestUpdater,
	memberships membershipChecker,
	jobEnqueue *JobService,
) *PaymentService {
	return &PaymentService{
		pool:        pool,
		payments:    payments,
		bookings:    bookings,
		events:      events,
		seats:       seats,
		guests:      guests,
		memberships: memberships,
		jobEnqueue:  jobEnqueue,
	}
}

type CreatePaymentInput struct {
	Amount     string
	Currency   string
	Method     *string
	GatewayRef *string
	Status     model.PaymentStatus
}

func isValidPaymentStatus(status model.PaymentStatus) bool {
	switch status {
	case model.PaymentPending, model.PaymentSuccess, model.PaymentFailed, model.PaymentRefunded:
		return true
	default:
		return false
	}
}

func isBookingPayable(status model.BookingStatus) bool {
	return status == model.BookingPending || status == model.BookingNotPaid
}

func parsePositiveAmount(amount string) (string, error) {
	trimmed := strings.TrimSpace(amount)
	if trimmed == "" {
		return "", ErrInvalidPaymentAmount
	}

	parsed, err := parseDecimal(trimmed)
	if err != nil {
		return "", ErrInvalidPaymentAmount
	}
	if parsed <= 0 {
		return "", ErrInvalidPaymentAmount
	}

	return trimmed, nil
}

func parseDecimal(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	if err != nil {
		return 0, err
	}
	return f, nil
}

func (s *PaymentService) Create(ctx context.Context, actorID, bookingID uuid.UUID, in CreatePaymentInput) (*model.Payment, error) {
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

	can, err := s.memberships.IsMember(ctx, s.pool, actorID, event.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !can {
		return nil, ErrForbidden
	}

	amount, err := parsePositiveAmount(in.Amount)
	if err != nil {
		return nil, err
	}

	currency := strings.TrimSpace(in.Currency)
	if currency == "" {
		currency = "IDR"
	}
	if !currencyRegex.MatchString(currency) {
		return nil, ErrInvalidCurrency
	}

	status := in.Status
	if status == "" {
		status = model.PaymentPending
	}
	if !isValidPaymentStatus(status) {
		return nil, ErrInvalidPaymentStatus
	}

	if status == model.PaymentSuccess && !isBookingPayable(booking.Status) {
		return nil, ErrBookingNotPayable
	}

	wasPaid := booking.Status == model.BookingPaid

	payment := &model.Payment{
		BookingID:  bookingID,
		Amount:     amount,
		Currency:   currency,
		Method:     in.Method,
		GatewayRef: in.GatewayRef,
		Status:     status,
	}

	if status == model.PaymentSuccess {
		now := time.Now()
		payment.PaidAt = &now
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := s.payments.Create(ctx, tx, payment)
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	if status == model.PaymentSuccess {
		booking.Status = model.BookingPaid
		if booking.PaidAt == nil {
			booking.PaidAt = payment.PaidAt
		}
		updatedBy := actorID
		booking.UpdatedBy = &updatedBy

		booking, err = ensureBookingBarcode(ctx, tx, s.bookings, booking)
		if err != nil {
			return nil, err
		}
		if err := s.seats.UpdateStatus(ctx, tx, booking.SeatID, model.SeatOccupied); err != nil {
			return nil, fmt.Errorf("update seat status: %w", err)
		}

		guest, err := s.guests.GetByID(ctx, tx, booking.GuestID)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrGuestNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("get guest: %w", err)
		}
		if guest.PaidDate == nil && payment.PaidAt != nil {
			guest.PaidDate = payment.PaidAt
			if _, err := s.guests.Update(ctx, tx, guest); err != nil {
				return nil, fmt.Errorf("update guest paid date: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	if status == model.PaymentSuccess && !wasPaid {
		s.enqueueInvitation(ctx, booking)
	}

	return created, nil
}

func (s *PaymentService) enqueueInvitation(ctx context.Context, booking *model.SeatBooking) {
	if s.jobEnqueue == nil {
		return
	}
	_, err := s.jobEnqueue.Enqueue(ctx, jobtype.SendInvitation, jobtype.SendInvitationPayload{
		BookingID: booking.ID,
		GuestID:   booking.GuestID,
		EventID:   booking.EventID,
	})
	if err != nil {
		_ = err
	}
}

func (s *PaymentService) ListByBooking(ctx context.Context, actorID, bookingID uuid.UUID) ([]model.Payment, error) {
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

	list, err := s.payments.ListByBookingID(ctx, s.pool, bookingID)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	return list, nil
}
