package invitation

import (
	"strings"
	"testing"
)

func TestRenderWithBarcodeIncludesQR(t *testing.T) {
	cfg := DefaultConfig()
	ctx := Context{
		GuestName:  "Alex",
		EventName:  "Gala Night",
		SeatCode:   "A-12",
		TicketCode: "EVVY-deadbeef1234",
	}

	result, err := Render(cfg, ctx, false)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(result.HTML, `src="cid:ticket-barcode"`) {
		t.Fatalf("expected cid image in html, got %q", result.HTML)
	}
	if !strings.Contains(result.HTML, "EVVY-deadbeef1234") {
		t.Fatalf("expected barcode in html, got %q", result.HTML)
	}
	if len(result.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(result.Attachments))
	}
	if result.Attachments[0].ContentID != QRContentID {
		t.Fatalf("unexpected content id %q", result.Attachments[0].ContentID)
	}
}

func TestRenderWithoutBarcodeNoQR(t *testing.T) {
	cfg := DefaultConfig()
	ctx := Context{
		GuestName:  "Alex",
		EventName:  "Gala Night",
		SeatCode:   "A-12",
		TicketCode: "",
	}

	result, err := Render(cfg, ctx, false)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if strings.Contains(result.HTML, "cid:ticket-barcode") {
		t.Fatalf("did not expect cid image in html, got %q", result.HTML)
	}
	if len(result.Attachments) != 0 {
		t.Fatalf("expected no attachments, got %d", len(result.Attachments))
	}
}

func TestRenderPreviewUsesDataURI(t *testing.T) {
	cfg := DefaultConfig()
	ctx := SampleContext()

	result, err := Render(cfg, ctx, true)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(result.HTML, "data:image/png;base64,") {
		t.Fatalf("expected data uri qr in preview html")
	}
	if len(result.Attachments) != 0 {
		t.Fatalf("preview should not attach files")
	}
}

func TestConfigValidateRequiresBannerWhenEnabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BannerEnabled = true

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when banner enabled without url")
	}
}

func TestApplyMergeTagsStripsUnknown(t *testing.T) {
	got := applyMergeTags("Hello {{guest_name}} {{unknown}}", Context{GuestName: "Ada"})
	if strings.Contains(got, "{{unknown}}") {
		t.Fatalf("unknown tag leaked: %q", got)
	}
	if !strings.Contains(got, "Ada") {
		t.Fatalf("expected guest name in output: %q", got)
	}
}

func TestRenderEmptyGreetingOmitted(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Greeting = ""
	ctx := SampleContext()

	result, err := Render(cfg, ctx, true)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if strings.Contains(result.HTML, "Hi Ada Lovelace") {
		t.Fatalf("expected empty greeting to be omitted, got %q", result.HTML)
	}
}

func TestRenderEmptyQRAndTicketLabelsOmitted(t *testing.T) {
	cfg := DefaultConfig()
	cfg.QRLabel = ""
	cfg.TicketCodeLabel = ""
	ctx := SampleContext()

	result, err := Render(cfg, ctx, false)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if strings.Contains(result.HTML, "Scan this code at the door:") {
		t.Fatalf("expected qr label omitted")
	}
	if strings.Contains(result.HTML, "Ticket code:") {
		t.Fatalf("expected ticket label omitted")
	}
	if !strings.Contains(result.HTML, `src="cid:ticket-barcode"`) {
		t.Fatalf("expected qr image")
	}
	if !strings.Contains(result.HTML, "EVVY-deadbeef1234") {
		t.Fatalf("expected ticket code")
	}
}

func TestSanitizeBodyHTMLAllowsListsAndEmphasis(t *testing.T) {
	raw := `<p>Hello</p><em>italic</em><ul><li>one</li></ul><ol><li>two</li></ol>`
	got := sanitizeBodyHTML(raw)
	if got != raw {
		t.Fatalf("sanitizeBodyHTML() = %q, want %q", got, raw)
	}
}

func TestRenderHeadlinePlainTextNotColoredBar(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Headline = "THE MAGIC IS CALLING YOU"
	ctx := SampleContext()

	result, err := Render(cfg, ctx, true)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if strings.Contains(result.HTML, "background:#1a1a2e") {
		t.Fatalf("headline should not use colored background bar")
	}
	if !strings.Contains(result.HTML, "THE MAGIC IS CALLING YOU") {
		t.Fatalf("expected headline in html")
	}
}
