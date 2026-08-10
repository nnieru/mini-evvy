package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/jobtype"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
)

const InvitationResendCooldown = 15 * time.Minute

var (
	ErrInvitationResendTooSoon   = errors.New("invitation resend too soon")
	ErrInvitationSendInProgress  = errors.New("invitation send in progress")
)

type InvitationResendCooldownError struct {
	AvailableAt time.Time
}

func (e *InvitationResendCooldownError) Error() string {
	return fmt.Sprintf("resend available after %s", e.AvailableAt.UTC().Format(time.RFC3339))
}

func (e *InvitationResendCooldownError) Is(target error) bool {
	return target == ErrInvitationResendTooSoon
}

func InvitationResendAvailableAt(
	status model.InvitationEmailStatus,
	sentAt *time.Time,
	now time.Time,
) *time.Time {
	if status != model.InvitationEmailSent || sentAt == nil {
		return nil
	}
	available := sentAt.Add(InvitationResendCooldown)
	if now.Before(available) {
		return &available
	}
	return nil
}

func validateInvitationResend(booking *model.SeatBooking, now time.Time) error {
	if booking.InvitationEmailStatus == model.InvitationEmailPending {
		return ErrInvitationSendInProgress
	}
	if available := InvitationResendAvailableAt(booking.InvitationEmailStatus, booking.InvitationEmailSentAt, now); available != nil {
		return &InvitationResendCooldownError{AvailableAt: *available}
	}
	return nil
}

type invitationEmailStatusStore interface {
	SetInvitationEmailPending(ctx context.Context, db repository.DBTX, id uuid.UUID) error
	UpdateInvitationEmailResult(
		ctx context.Context,
		db repository.DBTX,
		id uuid.UUID,
		status model.InvitationEmailStatus,
		sentAt *time.Time,
	) error
}

func revertInvitationEnqueueFailure(
	ctx context.Context,
	db repository.DBTX,
	bookings invitationEmailStatusStore,
	booking *model.SeatBooking,
) {
	status := model.InvitationEmailNotSent
	var sentAt *time.Time
	if booking.InvitationEmailSentAt != nil {
		status = model.InvitationEmailSent
		sentAt = booking.InvitationEmailSentAt
	}
	_ = bookings.UpdateInvitationEmailResult(ctx, db, booking.ID, status, sentAt)
}

func enqueueInvitationEmail(
	ctx context.Context,
	db repository.DBTX,
	bookings invitationEmailStatusStore,
	enqueue jobEnqueuer,
	booking *model.SeatBooking,
) (*model.Job, error) {
	if err := bookings.SetInvitationEmailPending(ctx, db, booking.ID); err != nil {
		return nil, fmt.Errorf("set invitation pending: %w", err)
	}

	job, err := enqueue.Enqueue(ctx, jobtype.SendInvitation, jobtype.SendInvitationPayload{
		BookingID: booking.ID,
		GuestID:   booking.GuestID,
		EventID:   booking.EventID,
	})
	if err != nil {
		revertInvitationEnqueueFailure(ctx, db, bookings, booking)
		return nil, fmt.Errorf("enqueue send invitation: %w", err)
	}

	return job, nil
}
