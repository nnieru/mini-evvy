package jobtype

import "github.com/google/uuid"

type FinalizeSeatingPayload struct {
	EventID uuid.UUID `json:"event_id"`
	ActorID uuid.UUID `json:"actor_id"`
}

type SendInvitationPayload struct {
	BookingID uuid.UUID `json:"booking_id"`
	GuestID   uuid.UUID `json:"guest_id"`
	EventID   uuid.UUID `json:"event_id"`
}
