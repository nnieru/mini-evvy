package service

import (
	"errors"

	"github.com/nnieru/mini-evvy/internal/model"
)

var (
	ErrSeatingLocked         = errors.New("seating is locked for review")
	ErrSeatingNotOpen        = errors.New("seating is not open for finalize")
	ErrSeatingNotPreview     = errors.New("seating is not in preview phase")
	ErrSeatingNotApproved    = errors.New("seating is not approved yet")
	ErrNoSeatingAssignments  = errors.New("no seat assignments could be made")
	ErrSeatingCapacityExhausted = errors.New("no available seats in the categories your guests need")
)

func ensureSeatingEditable(phase model.SeatingPhase) error {
	if phase == model.SeatingPreview {
		return ErrSeatingLocked
	}
	return nil
}
