package service

import (
	"testing"

	"github.com/google/uuid"
)

func TestSeatingReadinessFromDemand(t *testing.T) {
	vip := uuid.New()
	ga := uuid.New()

	t.Run("category full reserved blocks assign", func(t *testing.T) {
		demand := map[uuid.UUID]int{vip: 3}
		available := map[uuid.UUID]int{vip: 0}
		slots, canAssign, shortfalls := seatingReadinessFromDemand(demand, available)
		if slots != 3 || canAssign || len(shortfalls) != 1 {
			t.Fatalf("slots=%d canAssign=%v shortfalls=%d", slots, canAssign, len(shortfalls))
		}
	})

	t.Run("partial category capacity allows assign", func(t *testing.T) {
		demand := map[uuid.UUID]int{vip: 3, ga: 2}
		available := map[uuid.UUID]int{vip: 0, ga: 5}
		slots, canAssign, shortfalls := seatingReadinessFromDemand(demand, available)
		if slots != 5 || !canAssign || len(shortfalls) != 1 {
			t.Fatalf("slots=%d canAssign=%v shortfalls=%d", slots, canAssign, len(shortfalls))
		}
	})

	t.Run("no demand", func(t *testing.T) {
		slots, canAssign, shortfalls := seatingReadinessFromDemand(map[uuid.UUID]int{}, map[uuid.UUID]int{vip: 2})
		if slots != 0 || canAssign || len(shortfalls) != 0 {
			t.Fatalf("slots=%d canAssign=%v shortfalls=%d", slots, canAssign, len(shortfalls))
		}
	})
}
