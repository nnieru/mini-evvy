package dto

import (
	"github.com/nnieru/mini-evvy/internal/service"
)

type CategoryCapacityShortfallDTO struct {
	CategoryID     string `json:"category_id"`
	SlotsNeeded    int    `json:"slots_needed"`
	SeatsAvailable int    `json:"seats_available"`
}

type SeatingReadinessDTO struct {
	UnbookedGuests int                            `json:"unbooked_guests"`
	SlotsNeeded    int                            `json:"slots_needed"`
	CanAssignAny   bool                           `json:"can_assign_any"`
	Shortfalls     []CategoryCapacityShortfallDTO `json:"shortfalls"`
}

func NewSeatingReadinessDTO(r *service.SeatingReadiness) SeatingReadinessDTO {
	shortfalls := make([]CategoryCapacityShortfallDTO, 0, len(r.Shortfalls))
	for _, item := range r.Shortfalls {
		shortfalls = append(shortfalls, CategoryCapacityShortfallDTO{
			CategoryID:     item.CategoryID.String(),
			SlotsNeeded:    item.SlotsNeeded,
			SeatsAvailable: item.SeatsAvailable,
		})
	}
	return SeatingReadinessDTO{
		UnbookedGuests: r.UnbookedGuests,
		SlotsNeeded:    r.SlotsNeeded,
		CanAssignAny:   r.CanAssignAny,
		Shortfalls:     shortfalls,
	}
}
