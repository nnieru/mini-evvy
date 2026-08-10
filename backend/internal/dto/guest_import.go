package dto

import "github.com/nnieru/mini-evvy/internal/service"

type GuestImportRowErrorDTO struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

type GuestImportResultDTO struct {
	Created int                      `json:"created"`
	Updated int                      `json:"updated"`
	Skipped int                      `json:"skipped"`
	Failed  int                      `json:"failed"`
	Errors  []GuestImportRowErrorDTO `json:"errors"`
}

func NewGuestImportResultDTO(r *service.GuestImportResult) GuestImportResultDTO {
	errs := make([]GuestImportRowErrorDTO, 0, len(r.Errors))
	for _, e := range r.Errors {
		errs = append(errs, GuestImportRowErrorDTO{Row: e.Row, Message: e.Message})
	}
	return GuestImportResultDTO{
		Created: r.Created,
		Updated: r.Updated,
		Skipped: r.Skipped,
		Failed:  r.Failed,
		Errors:  errs,
	}
}
