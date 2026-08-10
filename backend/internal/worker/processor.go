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
)

type emailTemplateLoader interface {
	GetByEventAndType(ctx context.Context, db repository.DBTX, eventID uuid.UUID, templateType string) (*model.EventEmailTemplate, error)
}

type seatingDraftWorkerStore interface {
	Create(ctx context.Context, db repository.DBTX, draft *model.SeatingDraft) (*model.SeatingDraft, error)
	GetOpenByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) (*model.SeatingDraft, error)
	HasApprovedByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) (bool, error)
	CreateItem(ctx context.Context, db repository.DBTX, item *model.SeatingDraftItem) (*model.SeatingDraftItem, error)
	CountItemsByGuestID(ctx context.Context, db repository.DBTX, draftID, guestID uuid.UUID) (int, error)
	ListItemsByDraftID(ctx context.Context, db repository.DBTX, draftID uuid.UUID) ([]model.SeatingDraftItem, error)
	UpdateStatus(ctx context.Context, db repository.DBTX, draftID uuid.UUID, status model.SeatingDraftStatus) error
}

type seatingDraftSeatWorkerStore interface {
	GetByID(ctx context.Context, db repository.DBTX, id uuid.UUID) (*model.Seat, error)
	ListByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID, status *model.SeatStatus, categoryID *uuid.UUID) ([]model.Seat, error)
	ClaimHeldFromAvailable(ctx context.Context, db repository.DBTX, seatID uuid.UUID) error
	ReleaseHeld(ctx context.Context, db repository.DBTX, seatID uuid.UUID) error
}

const maxRetries = 3

type Processor struct {
	pool       *pgxpool.Pool
	jobs       *repository.JobRepo
	guests     *repository.GuestRepo
	seats      seatingDraftSeatWorkerStore
	bookings   *repository.BookingRepo
	events     *repository.EventRepo
	drafts     seatingDraftWorkerStore
	templates  emailTemplateLoader
	jobEnqueue *service.JobService
	mailer     mailer.Mailer
}

func NewProcessor(
	pool *pgxpool.Pool,
	jobs *repository.JobRepo,
	guests *repository.GuestRepo,
	seats seatingDraftSeatWorkerStore,
	bookings *repository.BookingRepo,
	events *repository.EventRepo,
	drafts seatingDraftWorkerStore,
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
		drafts:     drafts,
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
	var draftID uuid.UUID
	defer func() {
		if failed && draftID != uuid.Nil {
			p.cleanupFailedDraft(ctx, draftID)
		}
		if failed {
			hasApproved, err := p.drafts.HasApprovedByEventID(ctx, p.pool, payload.EventID)
			if err != nil {
				hasApproved = false
			}
			nextPhase := model.SeatingOpen
			if hasApproved {
				nextPhase = model.SeatingApproved
			}
			if err := p.events.UpdateSeatingPhase(ctx, p.pool, payload.EventID, nextPhase); err != nil {
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

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		failed = true
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	draft, err := p.drafts.GetOpenByEventID(ctx, tx, payload.EventID)
	if errors.Is(err, repository.ErrNotFound) {
		draft, err = p.drafts.Create(ctx, tx, &model.SeatingDraft{
			EventID:   payload.EventID,
			Status:    model.SeatingDraftOpen,
			CreatedBy: payload.ActorID,
		})
	}
	if err != nil {
		failed = true
		return fmt.Errorf("ensure open draft: %w", err)
	}
	draftID = draft.ID

	var assigned int
	var shortfalls int

	for _, guest := range guests {
		activeCount, err := p.bookings.CountActiveByGuestID(ctx, tx, guest.ID)
		if err != nil {
			failed = true
			return fmt.Errorf("count active bookings: %w", err)
		}
		draftCount, err := p.drafts.CountItemsByGuestID(ctx, tx, draft.ID, guest.ID)
		if err != nil {
			failed = true
			return fmt.Errorf("count draft items: %w", err)
		}
		slotsNeeded := guest.TicketCount - activeCount - draftCount
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

			if err := p.seats.ClaimHeldFromAvailable(ctx, tx, seat.ID); err != nil {
				if errors.Is(err, repository.ErrSeatNotAvailable) {
					shortfalls++
					continue
				}
				failed = true
				return fmt.Errorf("hold seat: %w", err)
			}

			_, err := p.drafts.CreateItem(ctx, tx, &model.SeatingDraftItem{
				DraftID:    draft.ID,
				GuestID:    guest.ID,
				SeatID:     seat.ID,
				CategoryID: seat.CategoryID,
			})
			if err != nil {
				failed = true
				return fmt.Errorf("create draft item: %w", err)
			}
			assigned++
		}
	}

	draftItems, err := p.drafts.ListItemsByDraftID(ctx, tx, draft.ID)
	if err != nil {
		failed = true
		return fmt.Errorf("list draft items: %w", err)
	}
	if len(draftItems) == 0 {
		hasApproved, approveErr := p.drafts.HasApprovedByEventID(ctx, tx, payload.EventID)
		if approveErr != nil {
			failed = true
			return fmt.Errorf("check approved drafts: %w", approveErr)
		}
		if err := p.drafts.UpdateStatus(ctx, tx, draft.ID, model.SeatingDraftRejected); err != nil {
			failed = true
			return fmt.Errorf("reject empty draft: %w", err)
		}
		nextPhase := model.SeatingOpen
		if hasApproved {
			nextPhase = model.SeatingApproved
		}
		if err := p.events.UpdateSeatingPhase(ctx, tx, payload.EventID, nextPhase); err != nil {
			failed = true
			return fmt.Errorf("revert seating phase: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			failed = true
			return fmt.Errorf("commit transaction: %w", err)
		}

		return fmt.Errorf(
			"%w: %d unbooked guests, %d available seats, %d seat shortages",
			service.ErrNoSeatingAssignments,
			len(guests),
			len(seatList),
			shortfalls,
		)
	}

	if err := p.events.UpdateSeatingPhase(ctx, tx, payload.EventID, model.SeatingPreview); err != nil {
		failed = true
		return fmt.Errorf("set seating preview: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		failed = true
		return fmt.Errorf("commit transaction: %w", err)
	}

	slog.Info("finalize seating complete",
		"event_id", payload.EventID,
		"draft_id", draftID,
		"assigned", assigned,
		"shortfalls", shortfalls,
	)

	return nil
}

func (p *Processor) cleanupFailedDraft(ctx context.Context, draftID uuid.UUID) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)

	items, err := p.drafts.ListItemsByDraftID(ctx, tx, draftID)
	if err != nil {
		return
	}
	for _, item := range items {
		_ = p.seats.ReleaseHeld(ctx, tx, item.SeatID)
	}
	_ = p.drafts.UpdateStatus(ctx, tx, draftID, model.SeatingDraftRejected)
	_ = tx.Commit(ctx)
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
	if event.SeatingPhase != model.SeatingApproved {
		slog.Info("skip invitation: seating not approved", "event_id", payload.EventID, "phase", event.SeatingPhase)
		return nil
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
		slog.Error("resend send failed", "job_type", jobtype.SendInvitation, "to", guest.Email, "error", err)
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
