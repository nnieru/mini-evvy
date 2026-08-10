package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
)

type CategoryCapacityShortfall struct {
	CategoryID     uuid.UUID
	SlotsNeeded    int
	SeatsAvailable int
}

type SeatingReadiness struct {
	UnbookedGuests int
	SlotsNeeded    int
	CanAssignAny   bool
	Shortfalls     []CategoryCapacityShortfall
}

func seatingReadinessFromDemand(
	demand map[uuid.UUID]int,
	available map[uuid.UUID]int,
) (slotsNeeded int, canAssignAny bool, shortfalls []CategoryCapacityShortfall) {
	for _, needed := range demand {
		slotsNeeded += needed
	}

	categoryIDs := make([]uuid.UUID, 0, len(demand))
	for categoryID := range demand {
		categoryIDs = append(categoryIDs, categoryID)
	}
	sort.Slice(categoryIDs, func(i, j int) bool {
		return categoryIDs[i].String() < categoryIDs[j].String()
	})

	for _, categoryID := range categoryIDs {
		needed := demand[categoryID]
		if needed <= 0 {
			continue
		}
		avail := available[categoryID]
		if avail > 0 {
			canAssignAny = true
		}
		if needed > avail {
			shortfalls = append(shortfalls, CategoryCapacityShortfall{
				CategoryID:     categoryID,
				SlotsNeeded:    needed,
				SeatsAvailable: avail,
			})
		}
	}

	return slotsNeeded, canAssignAny, shortfalls
}

func (s *FinalizeService) GetSeatingReadiness(ctx context.Context, actorID, eventID uuid.UUID) (*SeatingReadiness, error) {
	if _, err := s.ensureOwnerAdmin(ctx, actorID, eventID); err != nil {
		return nil, err
	}

	readiness, err := s.computeSeatingReadiness(ctx, eventID)
	if err != nil {
		return nil, err
	}
	return readiness, nil
}

func (s *FinalizeService) computeSeatingReadiness(ctx context.Context, eventID uuid.UUID) (*SeatingReadiness, error) {
	guests, err := s.guests.ListUnbookedByEventID(ctx, s.pool, eventID)
	if err != nil {
		return nil, fmt.Errorf("list unbooked guests: %w", err)
	}

	var openDraftID *uuid.UUID
	if draft, err := s.drafts.GetOpenByEventID(ctx, s.pool, eventID); err == nil {
		openDraftID = &draft.ID
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("get open draft: %w", err)
	}

	demand := make(map[uuid.UUID]int)
	unbookedGuests := 0

	for _, guest := range guests {
		activeCount, err := s.bookings.CountActiveByGuestID(ctx, s.pool, guest.ID)
		if err != nil {
			return nil, fmt.Errorf("count active bookings: %w", err)
		}
		draftCount := 0
		if openDraftID != nil {
			draftCount, err = s.drafts.CountItemsByGuestID(ctx, s.pool, *openDraftID, guest.ID)
			if err != nil {
				return nil, fmt.Errorf("count draft items: %w", err)
			}
		}
		slotsNeeded := guest.TicketCount - activeCount - draftCount
		if slotsNeeded <= 0 {
			continue
		}
		unbookedGuests++
		demand[guest.CategoryID] += slotsNeeded
	}

	available := model.SeatAvailable
	seatList, err := s.seats.ListByEventID(ctx, s.pool, eventID, &available, nil)
	if err != nil {
		return nil, fmt.Errorf("list available seats: %w", err)
	}

	availByCategory := make(map[uuid.UUID]int, len(seatList))
	for _, seat := range seatList {
		availByCategory[seat.CategoryID]++
	}

	totalSlots, canAssignAny, shortfalls := seatingReadinessFromDemand(demand, availByCategory)

	return &SeatingReadiness{
		UnbookedGuests: unbookedGuests,
		SlotsNeeded:    totalSlots,
		CanAssignAny:   canAssignAny,
		Shortfalls:     shortfalls,
	}, nil
}
