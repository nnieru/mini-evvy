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
