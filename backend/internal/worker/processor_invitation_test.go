package worker

import (
	"strings"
	"testing"

	"github.com/nnieru/mini-evvy/internal/mailer/invitation"
	"github.com/nnieru/mini-evvy/internal/ticket"
)

func TestInvitationRenderWithBarcode(t *testing.T) {
	cfg := invitation.DefaultConfig()
	result, err := invitation.Render(cfg, invitation.Context{
		GuestName:  "Alex",
		EventName:  "Gala Night",
		SeatCode:   "A-12",
		TicketCode: "EVVY-deadbeef1234",
	}, false)
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
	if result.Attachments[0].ContentID != invitation.QRContentID {
		t.Fatalf("unexpected content id %q", result.Attachments[0].ContentID)
	}
}

func TestInvitationRenderWithoutBarcode(t *testing.T) {
	cfg := invitation.DefaultConfig()
	result, err := invitation.Render(cfg, invitation.Context{
		GuestName:  "Alex",
		EventName:  "Gala Night",
		SeatCode:   "A-12",
		TicketCode: "",
	}, false)
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

func TestInvitationRenderQRMatchesBarcode(t *testing.T) {
	barcode := "EVVY-cafebabe0001"
	cfg := invitation.DefaultConfig()
	result, err := invitation.Render(cfg, invitation.Context{
		GuestName:  "Alex",
		EventName:  "Gala Night",
		SeatCode:   "A-12",
		TicketCode: barcode,
	}, false)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	decoded, err := ticket.GenerateQRPNG(barcode)
	if err != nil {
		t.Fatalf("GenerateQRPNG() error = %v", err)
	}
	if len(result.Attachments[0].Content) != len(decoded) {
		t.Fatalf("qr png length mismatch")
	}
}
