package dto

import "github.com/nnieru/mini-evvy/internal/service"

type ImportEventConfigRequestDTO struct {
	SourceEventID        string `json:"source_event_id"`
	IncludeCategories    bool   `json:"include_categories"`
	IncludeSeats         bool   `json:"include_seats"`
	IncludeEmailTemplate bool   `json:"include_email_template"`
}

type ImportEventConfigResultDTO struct {
	CategoriesCreated   int  `json:"categories_created"`
	SeatsCreated        int  `json:"seats_created"`
	EmailTemplateCopied bool `json:"email_template_copied"`
}

func NewImportEventConfigResultDTO(result *service.ImportConfigResult) ImportEventConfigResultDTO {
	return ImportEventConfigResultDTO{
		CategoriesCreated:   result.CategoriesCreated,
		SeatsCreated:        result.SeatsCreated,
		EmailTemplateCopied: result.EmailTemplateCopied,
	}
}
