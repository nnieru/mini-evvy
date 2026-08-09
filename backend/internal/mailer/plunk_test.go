package mailer

import (
	"encoding/json"
	"testing"
)

func TestParsePlunkFrom(t *testing.T) {
	t.Run("name and email", func(t *testing.T) {
		got, err := parsePlunkFrom("Mini Evvy <noreply@evvy.fun>")
		if err != nil {
			t.Fatalf("parsePlunkFrom() error = %v", err)
		}
		from, ok := got.(plunkFrom)
		if !ok {
			t.Fatalf("expected plunkFrom, got %T", got)
		}
		if from.Name != "Mini Evvy" || from.Email != "noreply@evvy.fun" {
			t.Fatalf("unexpected from %#v", from)
		}
	})

	t.Run("plain email", func(t *testing.T) {
		got, err := parsePlunkFrom("noreply@evvy.fun")
		if err != nil {
			t.Fatalf("parsePlunkFrom() error = %v", err)
		}
		email, ok := got.(string)
		if !ok || email != "noreply@evvy.fun" {
			t.Fatalf("expected plain email string, got %#v", got)
		}
	})
}

func TestToPlunkAttachmentsInlineCID(t *testing.T) {
	attachments := toPlunkAttachments([]Attachment{
		{
			Filename:    "ticket.png",
			ContentType: "image/png",
			Content:     []byte("png"),
			ContentID:   "ticket-barcode",
		},
	})

	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	if attachments[0].ContentID != "ticket-barcode" {
		t.Fatalf("unexpected contentId %q", attachments[0].ContentID)
	}
	if attachments[0].Disposition != "inline" {
		t.Fatalf("expected inline disposition, got %q", attachments[0].Disposition)
	}

	raw, err := json.Marshal(attachments[0])
	if err != nil {
		t.Fatalf("marshal attachment: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("invalid json: %s", string(raw))
	}
}

func TestToPlunkAttachmentsRegularFile(t *testing.T) {
	attachments := toPlunkAttachments([]Attachment{
		{
			Filename:    "invoice.pdf",
			ContentType: "application/pdf",
			Content:     []byte("pdf"),
		},
	})

	if attachments[0].Disposition != "" {
		t.Fatalf("expected empty disposition, got %q", attachments[0].Disposition)
	}
	if attachments[0].ContentID != "" {
		t.Fatalf("expected empty contentId, got %q", attachments[0].ContentID)
	}
}
