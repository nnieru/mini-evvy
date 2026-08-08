package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
)

func (s *BookingService) findOrCreateGuest(
	ctx context.Context,
	db repository.DBTX,
	eventID, categoryID uuid.UUID,
	name, email string,
	ticketDelta int,
) (*model.Guest, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	existing, err := s.guests.GetByEventNameEmailCategory(ctx, db, eventID, categoryID, name, email)
	if errors.Is(err, repository.ErrNotFound) {
		created, err := s.guests.Create(ctx, db, &model.Guest{
			EventID:     eventID,
			CategoryID:  categoryID,
			Name:        name,
			Email:       email,
			TicketCount: ticketDelta,
		})
		if err != nil {
			return nil, fmt.Errorf("create guest: %w", err)
		}
		return created, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lookup guest: %w", err)
	}

	existing.TicketCount += ticketDelta
	updated, err := s.guests.Update(ctx, db, existing)
	if err != nil {
		return nil, fmt.Errorf("update guest ticket count: %w", err)
	}
	return updated, nil
}
