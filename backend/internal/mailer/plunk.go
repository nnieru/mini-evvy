package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
)

const plunkSendURL = "https://next-api.useplunk.com/v1/send"

type PlunkMailer struct {
	apiKey string
	from   string
	client *http.Client
}

func NewPlunkMailer(apiKey, from string) *PlunkMailer {
	return &PlunkMailer{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{},
	}
}

type plunkFrom struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

type plunkRequest struct {
	To          string            `json:"to"`
	From        any               `json:"from"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	Attachments []plunkAttachment `json:"attachments,omitempty"`
}

// parsePlunkFrom converts EMAIL_FROM into Plunk's accepted shape:
// plain email string, or {name,email} when "Name <email>" is used.
func parsePlunkFrom(from string) (any, error) {
	from = strings.TrimSpace(from)
	if from == "" {
		return nil, fmt.Errorf("from address required")
	}

	addr, err := mail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("invalid from address: %w", err)
	}

	if strings.TrimSpace(addr.Name) == "" {
		return addr.Address, nil
	}

	return plunkFrom{Name: addr.Name, Email: addr.Address}, nil
}

type plunkAttachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	ContentType string `json:"contentType"`
	ContentID   string `json:"contentId,omitempty"`
	Disposition string `json:"disposition,omitempty"`
}

func toPlunkAttachments(attachments []Attachment) []plunkAttachment {
	out := make([]plunkAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		item := plunkAttachment{
			Filename:    attachment.Filename,
			Content:     base64.StdEncoding.EncodeToString(attachment.Content),
			ContentType: attachment.ContentType,
		}
		if attachment.ContentID != "" {
			item.ContentID = attachment.ContentID
			item.Disposition = "inline"
		}
		out = append(out, item)
	}
	return out
}

func (m *PlunkMailer) Send(ctx context.Context, to, subject, text, html string, attachments ...Attachment) error {
	if m.apiKey == "" {
		return fmt.Errorf("plunk api key not configured")
	}

	bodyHTML := html
	if bodyHTML == "" {
		bodyHTML = text
	}

	from, err := parsePlunkFrom(m.from)
	if err != nil {
		return err
	}

	body, err := json.Marshal(plunkRequest{
		To:          to,
		From:        from,
		Subject:     subject,
		Body:        bodyHTML,
		Attachments: toPlunkAttachments(attachments),
	})
	if err != nil {
		return fmt.Errorf("marshal plunk request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, plunkSendURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create plunk request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("send plunk request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plunk api error: status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
