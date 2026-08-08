package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, text, html string, attachments ...Attachment) error
}

type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
	ContentID   string
}

type ResendMailer struct {
	apiKey string
	from   string
	client *http.Client
}

func NewResendMailer(apiKey, from string) *ResendMailer {
	return &ResendMailer{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{},
	}
}

type resendRequest struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Subject     string             `json:"subject"`
	Text        string             `json:"text,omitempty"`
	HTML        string             `json:"html"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

type resendAttachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	ContentType string `json:"content_type,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

func (m *ResendMailer) Send(ctx context.Context, to, subject, text, html string, attachments ...Attachment) error {
	if m.apiKey == "" {
		return fmt.Errorf("resend api key not configured")
	}

	var resendAttachments []resendAttachment
	for _, attachment := range attachments {
		resendAttachments = append(resendAttachments, resendAttachment{
			Filename:    attachment.Filename,
			Content:     base64.StdEncoding.EncodeToString(attachment.Content),
			ContentType: attachment.ContentType,
			ContentID:   attachment.ContentID,
		})
	}

	body, err := json.Marshal(resendRequest{
		From:        m.from,
		To:          []string{to},
		Subject:     subject,
		Text:        text,
		HTML:        html,
		Attachments: resendAttachments,
	})
	if err != nil {
		return fmt.Errorf("marshal resend request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("send resend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend api error: status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
