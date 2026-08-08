package service

import (
	"errors"

	"github.com/nnieru/mini-evvy/internal/model"
)

var (
	ErrSeatingLocked     = errors.New("seating is locked for review")
	ErrSeatingNotOpen    = errors.New("seating is not open for finalize")
	ErrSeatingNotPreview = errors.New("seating is not in preview phase")
)

func ensureSeatingEditable(phase model.SeatingPhase) error {
	if phase == model.SeatingPreview {
		return ErrSeatingLocked
	}
	return nil
}
