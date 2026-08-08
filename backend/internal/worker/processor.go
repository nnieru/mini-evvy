package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/jobtype"
	"github.com/nnieru/mini-evvy/internal/mailer"
	"github.com/nnieru/mini-evvy/internal/mailer/invitation"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
	"github.com/nnieru/mini-evvy/internal/service"
	"github.com/nnieru/mini-evvy/internal/ticket"
)

type emailTemplateLoader interface {
	GetByEventAndType(ctx context.Context, db repository.DBTX, eventID uuid.UUID, templateType string) (*model.EventEmailTemplate, error)
}

const maxRetries = 3

type Processor struct {
	pool       *pgxpool.Pool
	jobs       *repository.JobRepo
	guests     *repository.GuestRepo
	seats      *repository.SeatRepo
	bookings   *repository.BookingRepo
	events     *repository.EventRepo
	templates  emailTemplateLoader
	jobEnqueue *service.JobService
	mailer     mailer.Mailer
}

func NewProcessor(
	pool *pgxpool.Pool,
	jobs *repository.JobRepo,
	guests *repository.GuestRepo,
	seats *repository.SeatRepo,
	bookings *repository.BookingRepo,
	events *repository.EventRepo,
	templates emailTemplateLoader,
	jobEnqueue *service.JobService,
	mailer mailer.Mailer,
) *Processor {
	return &Processor{
		pool:       pool,
		jobs:       jobs,
		guests:     guests,
		seats:      seats,
		bookings:   bookings,
		events:     events,
		templates:  templates,
		jobEnqueue: jobEnqueue,
		mailer:     mailer,
	}
}

func (p *Processor) Run(ctx context.Context, job *model.Job) error {
	switch job.Type {
	case jobtype.FinalizeSeating:
		return p.handleFinalizeSeating(ctx, job)
	case jobtype.SendInvitation:
		return p.handleSendInvitation(ctx, job)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

func (p *Processor) handleFinalizeSeating(ctx context.Context, job *model.Job) error {
	var payload jobtype.FinalizeSeatingPayload
	if err := json.Unmarshal(job.Data, &payload); err != nil {
		return fmt.Errorf("unmarshal finalize payload: %w", err)
	}

	var failed bool
	defer func() {
		if failed {
			if err := p.events.UpdateSeatingPhase(ctx, p.pool, payload.EventID, model.SeatingOpen); err != nil {
				slog.Error("revert seating phase failed", "event_id", payload.EventID, "error", err)
			}
		}
	}()

	guests, err := p.guests.ListUnbookedByEventID(ctx, p.pool, payload.EventID)
	if err != nil {
		failed = true
		return fmt.Errorf("list unbooked guests: %w", err)
	}

	available := model.SeatAvailable
	seatList, err := p.seats.ListByEventID(ctx, p.pool, payload.EventID, &available, nil)
	if err != nil {
		failed = true
		return fmt.Errorf("list available seats: %w", err)
	}

	pools := make(map[uuid.UUID][]model.Seat)
	for _, seat := range seatList {
		pools[seat.CategoryID] = append(pools[seat.CategoryID], seat)
	}

	var assigned int
	var shortfalls int

	for _, guest := range guests {
		activeCount, err := p.bookings.CountActiveByGuestID(ctx, p.pool, guest.ID)
		if err != nil {
			failed = true
			return fmt.Errorf("count active bookings: %w", err)
		}
		slotsNeeded := guest.TicketCount - activeCount
		if slotsNeeded <= 0 {
			continue
		}

		for i := 0; i < slotsNeeded; i++ {
			pool := pools[guest.CategoryID]
			if len(pool) == 0 {
				shortfalls++
				continue
			}
			seat := pool[0]
			pools[guest.CategoryID] = pool[1:]

			_, err := p.createAssignedBooking(ctx, payload.EventID, payload.ActorID, guest, seat)
			if err != nil {
				if errors.Is(err, repository.ErrSeatNotAvailable) {
					shortfalls++
					continue
				}
				failed = true
				return fmt.Errorf("create assigned booking: %w", err)
			}
			assigned++
		}
	}

	slog.Info("finalize seating complete",
		"event_id", payload.EventID,
		"assigned", assigned,
		"shortfalls", shortfalls,
	)

	if err := p.events.UpdateSeatingPhase(ctx, p.pool, payload.EventID, model.SeatingPreview); err != nil {
		failed = true
		return fmt.Errorf("set seating preview: %w", err)
	}

	return nil
}

func (p *Processor) createAssignedBooking(ctx context.Context, eventID, actorID uuid.UUID, guest model.Guest, seat model.Seat) (*model.SeatBooking, error) {
	if guest.CategoryID != seat.CategoryID {
		return nil, fmt.Errorf("create assigned booking: %w", service.ErrCategoryMismatch)
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	seatStatus := model.SeatReserved
	booking := &model.SeatBooking{
		GuestID:    guest.ID,
		EventID:    eventID,
		CategoryID: seat.CategoryID,
		SeatID:     seat.ID,
		Source:     model.SourceInvited,
		Status:     model.BookingPending,
		CreatedBy:  actorID,
	}
	if guest.PaidDate != nil {
		booking.Status = model.BookingPaid
		booking.PaidAt = guest.PaidDate
		seatStatus = model.SeatOccupied
	}

	if err := p.seats.ClaimFromAvailable(ctx, tx, seat.ID, seatStatus); err != nil {
		return nil, err
	}

	var created *model.SeatBooking
	for attempt := 0; attempt < 2; attempt++ {
		barcode, err := ticket.GenerateBarcode()
		if err != nil {
			return nil, err
		}
		booking.Barcode = &barcode

		created, err = p.bookings.Create(ctx, tx, booking)
		if err == nil {
			break
		}
		if isUniqueViolation(err) && attempt == 0 {
			continue
		}
		return nil, fmt.Errorf("create booking: %w", err)
	}
	if created == nil {
		return nil, fmt.Errorf("create booking failed after retries")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}

func (p *Processor) handleSendInvitation(ctx context.Context, job *model.Job) error {
	var payload jobtype.SendInvitationPayload
	if err := json.Unmarshal(job.Data, &payload); err != nil {
		return fmt.Errorf("unmarshal invitation payload: %w", err)
	}

	booking, err := p.bookings.GetByID(ctx, p.pool, payload.BookingID)
	if errors.Is(err, repository.ErrNotFound) {
		slog.Info("skip invitation: booking not found", "booking_id", payload.BookingID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("get booking: %w", err)
	}
	if booking.Status == model.BookingCancelled {
		slog.Info("skip invitation: booking cancelled", "booking_id", payload.BookingID)
		return nil
	}

	guest, err := p.guests.GetByID(ctx, p.pool, payload.GuestID)
	if errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("guest not found: %w", err)
	}
	if err != nil {
		return fmt.Errorf("get guest: %w", err)
	}
	if guest.Email == "" {
		slog.Info("skip invitation: guest email empty", "guest_id", payload.GuestID)
		return nil
	}

	event, err := p.events.GetByID(ctx, p.pool, payload.EventID)
	if errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("event not found: %w", err)
	}
	if err != nil {
		return fmt.Errorf("get event: %w", err)
	}

	seat, err := p.seats.GetByID(ctx, p.pool, booking.SeatID)
	if errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("seat not found: %w", err)
	}
	if err != nil {
		return fmt.Errorf("get seat: %w", err)
	}

	barcode := ""
	if booking.Barcode != nil {
		barcode = *booking.Barcode
	}

	cfg, err := service.LoadInvitationConfig(ctx, p.pool, p.templates, event.ID)
	if err != nil {
		return fmt.Errorf("load invitation template: %w", err)
	}

	rendered, err := invitation.Render(cfg, invitation.Context{
		GuestName:  guest.Name,
		EventName:  event.Name,
		SeatCode:   seat.Code,
		TicketCode: barcode,
	}, false)
	if err != nil {
		return fmt.Errorf("build invitation email: %w", err)
	}

	if err := p.mailer.Send(ctx, guest.Email, rendered.Subject, rendered.Text, rendered.HTML, rendered.Attachments...); err != nil {
		return fmt.Errorf("send invitation email: %w", err)
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
