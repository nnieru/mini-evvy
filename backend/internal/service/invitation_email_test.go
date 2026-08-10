package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/model"
)

func TestValidateInvitationResend(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	sentAt := now.Add(-5 * time.Minute)

	t.Run("pending blocked", func(t *testing.T) {
		booking := &model.SeatBooking{
			ID:                    uuid.New(),
			InvitationEmailStatus: model.InvitationEmailPending,
		}
		if err := validateInvitationResend(booking, now); err != ErrInvitationSendInProgress {
			t.Fatalf("expected ErrInvitationSendInProgress, got %v", err)
		}
	})

	t.Run("sent inside cooldown blocked", func(t *testing.T) {
		booking := &model.SeatBooking{
			ID:                    uuid.New(),
			InvitationEmailStatus: model.InvitationEmailSent,
			InvitationEmailSentAt: &sentAt,
		}
		err := validateInvitationResend(booking, now)
		var cooldown *InvitationResendCooldownError
		if !errorsAs(err, &cooldown) {
			t.Fatalf("expected cooldown error, got %v", err)
		}
	})

	t.Run("sent after cooldown allowed", func(t *testing.T) {
		oldSent := now.Add(-20 * time.Minute)
		booking := &model.SeatBooking{
			ID:                    uuid.New(),
			InvitationEmailStatus: model.InvitationEmailSent,
			InvitationEmailSentAt: &oldSent,
		}
		if err := validateInvitationResend(booking, now); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("failed allowed", func(t *testing.T) {
		booking := &model.SeatBooking{
			ID:                    uuid.New(),
			InvitationEmailStatus: model.InvitationEmailFailed,
		}
		if err := validateInvitationResend(booking, now); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}

func errorsAs(err error, target **InvitationResendCooldownError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*InvitationResendCooldownError); ok {
		*target = e
		return true
	}
	return false
}
