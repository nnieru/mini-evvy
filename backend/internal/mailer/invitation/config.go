package invitation

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const QRContentID = "ticket-barcode"

var (
	hexColorRe    = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	unknownTagRe  = regexp.MustCompile(`\{\{[^}]+\}\}`)
	allowedTagRe  = regexp.MustCompile(`</?(?:p|strong|br|a)(?:\s[^>]*)?>`)
)

type Config struct {
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
}

func DefaultConfig() Config {
	return Config{
		Subject:        "Your ticket — {{event_name}}",
		Headline:       "",
		Greeting:       "Hi {{guest_name}},",
		BodyHTML:       "",
		FooterText:     "Present this code at the door.",
		ShowQR:         true,
		ShowSeatCode:   true,
		ShowTicketCode: true,
		BannerEnabled:  false,
		BannerImageURL: nil,
		BannerAlt:      "",
		PrimaryColor:   "#1a1a2e",
	}
}

func ParseConfig(raw []byte) (Config, error) {
	if len(raw) == 0 {
		return DefaultConfig(), nil
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse invitation config: %w", err)
	}
	return NormalizeConfig(cfg), nil
}

func NormalizeConfig(cfg Config) Config {
	def := DefaultConfig()
	if strings.TrimSpace(cfg.Subject) == "" {
		cfg.Subject = def.Subject
	}
	if strings.TrimSpace(cfg.Greeting) == "" {
		cfg.Greeting = def.Greeting
	}
	if strings.TrimSpace(cfg.FooterText) == "" {
		cfg.FooterText = def.FooterText
	}
	if strings.TrimSpace(cfg.PrimaryColor) == "" {
		cfg.PrimaryColor = def.PrimaryColor
	}
	return cfg
}

func (c Config) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

func (c Config) Validate() error {
	if len(c.Subject) > 200 {
		return fmt.Errorf("subject must be at most 200 characters")
	}
	if len(c.Headline) > 200 {
		return fmt.Errorf("headline must be at most 200 characters")
	}
	if len(c.Greeting) > 500 {
		return fmt.Errorf("greeting must be at most 500 characters")
	}
	if len(c.BodyHTML) > 5000 {
		return fmt.Errorf("body must be at most 5000 characters")
	}
	if len(c.FooterText) > 1000 {
		return fmt.Errorf("footer must be at most 1000 characters")
	}
	if len(c.BannerAlt) > 200 {
		return fmt.Errorf("banner alt must be at most 200 characters")
	}
	if !hexColorRe.MatchString(c.PrimaryColor) {
		return fmt.Errorf("primary_color must be a hex color like #1a1a2e")
	}
	if c.BannerEnabled {
		if c.BannerImageURL == nil || strings.TrimSpace(*c.BannerImageURL) == "" {
			return fmt.Errorf("banner_image_url is required when banner is enabled")
		}
	}
	return nil
}

func (c Config) Sanitized() Config {
	out := c
	out.BodyHTML = sanitizeBodyHTML(out.BodyHTML)
	return out
}

func validateHTTPSURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("must be a valid URL")
	}
	if u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("must be an https URL")
	}
	return nil
}

func sanitizeBodyHTML(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var b strings.Builder
	rest := raw
	for {
		idx := strings.Index(rest, "<")
		if idx < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:idx])
		rest = rest[idx:]
		closeIdx := strings.Index(rest, ">")
		if closeIdx < 0 {
			b.WriteString(rest)
			break
		}
		tag := rest[:closeIdx+1]
		rest = rest[closeIdx+1:]
		lower := strings.ToLower(tag)
		if strings.HasPrefix(lower, "<a ") || lower == "<a>" {
			if href := extractHref(tag); href != "" {
				if u, err := url.Parse(href); err == nil && (u.Scheme == "https" || u.Scheme == "http" || u.Scheme == "mailto") {
					b.WriteString(fmt.Sprintf(`<a href="%s">`, htmlEscapeAttr(href)))
					continue
				}
			}
			continue
		}
		if allowedTagRe.MatchString(lower) {
			b.WriteString(tag)
		}
	}
	return b.String()
}

func extractHref(tag string) string {
	re := regexp.MustCompile(`(?i)href\s*=\s*("([^"]*)"|'([^']*)')`)
	m := re.FindStringSubmatch(tag)
	if len(m) < 3 {
		return ""
	}
	if m[2] != "" {
		return m[2]
	}
	return m[3]
}

func htmlEscapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
