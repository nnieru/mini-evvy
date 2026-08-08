package service

import (
	"context"
	"fmt"

	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
	"github.com/nnieru/mini-evvy/internal/ticket"
)

type bookingBarcodeUpdater interface {
	Update(ctx context.Context, db repository.DBTX, b *model.SeatBooking) (*model.SeatBooking, error)
}

func createBookingWithBarcode(
	ctx context.Context,
	db repository.DBTX,
	bookings bookingStore,
	booking *model.SeatBooking,
) (*model.SeatBooking, error) {
	for attempt := 0; attempt < 2; attempt++ {
		barcode, err := ticket.GenerateBarcode()
		if err != nil {
			return nil, err
		}
		booking.Barcode = &barcode

		created, err := bookings.Create(ctx, db, booking)
		if err == nil {
			return created, nil
		}
		if isUniqueViolation(err) && attempt == 0 {
			continue
		}
		return nil, fmt.Errorf("create booking: %w", err)
	}
	return nil, fmt.Errorf("create booking failed after retries")
}

func ensureBookingBarcode(
	ctx context.Context,
	db repository.DBTX,
	bookings bookingBarcodeUpdater,
	booking *model.SeatBooking,
) (*model.SeatBooking, error) {
	if booking.Barcode == nil || *booking.Barcode == "" {
		for attempt := 0; attempt < 2; attempt++ {
			barcode, err := ticket.GenerateBarcode()
			if err != nil {
				return nil, err
			}
			booking.Barcode = &barcode

			updated, err := bookings.Update(ctx, db, booking)
			if err == nil {
				return updated, nil
			}
			if isUniqueViolation(err) && attempt == 0 {
				continue
			}
			return nil, fmt.Errorf("update booking barcode: %w", err)
		}
		return nil, fmt.Errorf("update booking barcode failed after retries")
	}

	updated, err := bookings.Update(ctx, db, booking)
	if err != nil {
		return nil, fmt.Errorf("update booking: %w", err)
	}
	return updated, nil
}
