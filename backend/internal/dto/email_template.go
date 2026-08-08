package dto

import (
	"time"

	"github.com/nnieru/mini-evvy/internal/mailer/invitation"
	"github.com/nnieru/mini-evvy/internal/service"
)

type InvitationEmailTemplateDTO struct {
	Subject        string  `json:"subject"`
	Headline       string  `json:"headline"`
	Greeting       string  `json:"greeting"`
	BodyHTML       string  `json:"body_html"`
	FooterText     string  `json:"footer_text"`
	ShowQR         bool    `json:"show_qr"`
	ShowSeatCode   bool    `json:"show_seat_code"`
	ShowTicketCode bool    `json:"show_ticket_code"`
	BannerEnabled  bool    `json:"banner_enabled"`
	BannerImageURL *string `json:"banner_image_url"`
	BannerAlt      string  `json:"banner_alt"`
	PrimaryColor   string  `json:"primary_color"`
	IsDefault      bool    `json:"is_default"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type InvitationEmailPreviewDTO struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type BannerUploadResponseDTO struct {
	URL string `json:"url"`
}

func NewBannerUploadResponseDTO(url string) BannerUploadResponseDTO {
	return BannerUploadResponseDTO{URL: url}
}

func NewInvitationEmailTemplateDTO(view *service.EmailTemplateView) InvitationEmailTemplateDTO {
	return InvitationEmailTemplateDTO{
		Subject:        view.Config.Subject,
		Headline:       view.Config.Headline,
		Greeting:       view.Config.Greeting,
		BodyHTML:       view.Config.BodyHTML,
		FooterText:     view.Config.FooterText,
		ShowQR:         view.Config.ShowQR,
		ShowSeatCode:   view.Config.ShowSeatCode,
		ShowTicketCode: view.Config.ShowTicketCode,
		BannerEnabled:  view.Config.BannerEnabled,
		BannerImageURL: view.Config.BannerImageURL,
		BannerAlt:      view.Config.BannerAlt,
		PrimaryColor:   view.Config.PrimaryColor,
		IsDefault:      view.IsDefault,
		UpdatedAt:      view.UpdatedAt,
	}
}

func InvitationConfigFromDTO(d InvitationEmailTemplateDTO) invitation.Config {
	return invitation.Config{
		Subject:        d.Subject,
		Headline:       d.Headline,
		Greeting:       d.Greeting,
		BodyHTML:       d.BodyHTML,
		FooterText:     d.FooterText,
		ShowQR:         d.ShowQR,
		ShowSeatCode:   d.ShowSeatCode,
		ShowTicketCode: d.ShowTicketCode,
		BannerEnabled:  d.BannerEnabled,
		BannerImageURL: d.BannerImageURL,
		BannerAlt:      d.BannerAlt,
		PrimaryColor:   d.PrimaryColor,
	}
}

func NewInvitationEmailPreviewDTO(p *service.PreviewResult) InvitationEmailPreviewDTO {
	return InvitationEmailPreviewDTO{
		Subject: p.Subject,
		HTML:    p.HTML,
	}
}
