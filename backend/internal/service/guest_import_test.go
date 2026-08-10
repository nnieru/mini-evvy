package service

import (
	"testing"

	"github.com/nnieru/mini-evvy/internal/model"
)

func TestShouldSkipFullySeatedImportOnApprovedWave(t *testing.T) {
	t.Run("approved fully seated skips", func(t *testing.T) {
		if !shouldSkipFullySeatedImportOnApprovedWave(model.SeatingApproved, 2, 2) {
			t.Fatal("expected skip when active bookings equal ticket count")
		}
	})

	t.Run("approved partial shortfall merges", func(t *testing.T) {
		if shouldSkipFullySeatedImportOnApprovedWave(model.SeatingApproved, 1, 2) {
			t.Fatal("expected merge when guest still needs seats")
		}
	})

	t.Run("open phase fully seated still merges", func(t *testing.T) {
		if shouldSkipFullySeatedImportOnApprovedWave(model.SeatingOpen, 2, 2) {
			t.Fatal("expected merge on first wave even when fully seated")
		}
	})

	t.Run("approved overbooked skips", func(t *testing.T) {
		if !shouldSkipFullySeatedImportOnApprovedWave(model.SeatingApproved, 3, 2) {
			t.Fatal("expected skip when active bookings exceed ticket count")
		}
	})
}
