package invitation

import (
	"encoding/base64"
	"fmt"
	"html"
	"strings"

	"github.com/nnieru/mini-evvy/internal/mailer"
	"github.com/nnieru/mini-evvy/internal/ticket"
)

type Context struct {
	GuestName    string
	EventName    string
	SeatCode     string
	TicketCode   string
	UniqueSendID string // optional; breaks Gmail duplicate-body collapse when set
}

type RenderResult struct {
	Subject     string
	Text        string
	HTML        string
	Attachments []mailer.Attachment
}

func Render(cfg Config, ctx Context, preview bool) (RenderResult, error) {
	cfg = NormalizeConfig(cfg.Sanitized())

	subject := applyMergeTags(cfg.Subject, ctx)
	greeting := applyMergeTags(cfg.Greeting, ctx)
	headline := applyMergeTags(cfg.Headline, ctx)
	bodyHTML := applyMergeTags(cfg.BodyHTML, ctx)
	footer := applyMergeTags(cfg.FooterText, ctx)
	bannerAlt := applyMergeTags(cfg.BannerAlt, ctx)
	if strings.TrimSpace(bannerAlt) == "" {
		bannerAlt = html.EscapeString(ctx.EventName)
	}

	var parts []string
	parts = append(parts, `<!DOCTYPE html><html><body style="margin:0;padding:0;background:#f4f4f5;font-family:Arial,Helvetica,sans-serif;">`)
	parts = append(parts, fmt.Sprintf(
		`<div style="display:none;max-height:0;overflow:hidden;mso-hide:all;">%s</div>`,
		html.EscapeString(preheaderText(cfg, ctx)),
	))
	parts = append(parts, `<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:24px 0;"><tr><td align="center">`)
	parts = append(parts, `<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;background:#ffffff;border-radius:8px;overflow:hidden;">`)

	if cfg.BannerEnabled && cfg.BannerImageURL != nil && strings.TrimSpace(*cfg.BannerImageURL) != "" {
		parts = append(parts, fmt.Sprintf(
			`<tr><td><img src="%s" alt="%s" width="600" style="display:block;width:100%%;max-width:600px;height:auto;border:0;" /></td></tr>`,
			html.EscapeString(strings.TrimSpace(*cfg.BannerImageURL)),
			html.EscapeString(bannerAlt),
		))
	}

	if strings.TrimSpace(headline) != "" {
		parts = append(parts, fmt.Sprintf(
			`<tr><td style="padding:20px 24px 8px;background:%s;color:#ffffff;font-size:20px;font-weight:bold;">%s</td></tr>`,
			html.EscapeString(cfg.PrimaryColor),
			headline,
		))
	}

	contentStyle := "padding:24px;color:#111827;font-size:16px;line-height:1.5;"
	if strings.TrimSpace(headline) == "" {
		contentStyle = "padding:24px;color:#111827;font-size:16px;line-height:1.5;"
	}

	parts = append(parts, fmt.Sprintf(`<tr><td style="%s">`, contentStyle))
	if strings.TrimSpace(greeting) != "" {
		parts = append(parts, fmt.Sprintf(`<p style="margin:0 0 16px;">%s</p>`, greeting))
	}
	if strings.TrimSpace(bodyHTML) != "" {
		parts = append(parts, bodyHTML)
	}
	if cfg.ShowSeatCode {
		parts = append(parts, fmt.Sprintf(
			`<p style="margin:16px 0 0;">Your seat for <strong>%s</strong> is <strong>%s</strong>.</p>`,
			html.EscapeString(ctx.EventName),
			html.EscapeString(ctx.SeatCode),
		))
	}

	var attachments []mailer.Attachment
	if cfg.ShowQR && ctx.TicketCode != "" {
		qrPNG, err := ticket.GenerateQRPNG(ctx.TicketCode)
		if err != nil {
			return RenderResult{}, fmt.Errorf("generate qr: %w", err)
		}
		if preview {
			dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrPNG)
			parts = append(parts, fmt.Sprintf(
				`<p style="margin:16px 0 8px;">Scan this code at the door:</p><p style="margin:0 0 16px;"><img src="%s" alt="Ticket QR" width="200" height="200" style="display:block;" /></p>`,
				dataURI,
			))
		} else {
			parts = append(parts, fmt.Sprintf(
				`<p style="margin:16px 0 8px;">Scan this code at the door:</p><p style="margin:0 0 16px;"><img src="cid:%s" alt="Ticket QR" width="200" height="200" style="display:block;" /></p>`,
				QRContentID,
			))
			attachments = append(attachments, mailer.Attachment{
				Filename:    "ticket.png",
				ContentType: "image/png",
				Content:     qrPNG,
				ContentID:   QRContentID,
			})
		}
	}

	if cfg.ShowTicketCode && ctx.TicketCode != "" {
		parts = append(parts, `<p style="margin:16px 0 8px;">Ticket code:</p>`)
		parts = append(parts, fmt.Sprintf(
			`<pre style="margin:0 0 16px;padding:12px;background:#f3f4f6;border-radius:6px;font-size:14px;overflow-x:auto;">%s</pre>`,
			html.EscapeString(ctx.TicketCode),
		))
	}

	if strings.TrimSpace(footer) != "" {
		parts = append(parts, fmt.Sprintf(`<p style="margin:16px 0 0;color:#4b5563;">%s</p>`, footer))
	}

	parts = append(parts, `</td></tr></table></td></tr></table>`)
	if strings.TrimSpace(ctx.UniqueSendID) != "" {
		parts = append(parts, fmt.Sprintf(`<!-- send:%s -->`, html.EscapeString(ctx.UniqueSendID)))
	}
	parts = append(parts, `</body></html>`)

	return RenderResult{
		Subject:     subject,
		HTML:        strings.Join(parts, ""),
		Text:        RenderPlainText(cfg, ctx),
		Attachments: attachments,
	}, nil
}

func preheaderText(cfg Config, ctx Context) string {
	greeting := strings.TrimSpace(applyMergeTags(cfg.Greeting, ctx))
	if greeting != "" {
		return greeting
	}
	if cfg.ShowSeatCode {
		return fmt.Sprintf("Your seat for %s is %s.", ctx.EventName, ctx.SeatCode)
	}
	return fmt.Sprintf("Your ticket for %s", ctx.EventName)
}

func RenderPlainText(cfg Config, ctx Context) string {
	cfg = NormalizeConfig(cfg.Sanitized())
	var lines []string

	greeting := strings.TrimSpace(stripHTML(applyMergeTags(cfg.Greeting, ctx)))
	if greeting != "" {
		lines = append(lines, greeting)
	}
	body := strings.TrimSpace(stripHTML(applyMergeTags(cfg.BodyHTML, ctx)))
	if body != "" {
		lines = append(lines, body)
	}
	if cfg.ShowSeatCode {
		lines = append(lines, fmt.Sprintf("Your seat for %s is %s.", ctx.EventName, ctx.SeatCode))
	}
	if cfg.ShowTicketCode && ctx.TicketCode != "" {
		lines = append(lines, "Ticket code:", ctx.TicketCode)
	}
	footer := strings.TrimSpace(stripHTML(applyMergeTags(cfg.FooterText, ctx)))
	if footer != "" {
		lines = append(lines, footer)
	}
	return strings.Join(lines, "\n\n")
}

func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func applyMergeTags(s string, ctx Context) string {
	replacer := strings.NewReplacer(
		"{{guest_name}}", html.EscapeString(ctx.GuestName),
		"{{event_name}}", html.EscapeString(ctx.EventName),
		"{{seat_code}}", html.EscapeString(ctx.SeatCode),
		"{{ticket_code}}", html.EscapeString(ctx.TicketCode),
	)
	out := replacer.Replace(s)
	return unknownTagRe.ReplaceAllString(out, "")
}

func SampleContext() Context {
	return Context{
		GuestName:  "Ada Lovelace",
		EventName:  "Gala Night",
		SeatCode:   "A12",
		TicketCode: "EVVY-deadbeef1234",
	}
}
