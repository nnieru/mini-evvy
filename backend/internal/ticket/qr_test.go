package ticket

import (
	"bytes"
	"image/png"
	"testing"
)

func TestGenerateQRPNG(t *testing.T) {
	t.Run("valid barcode", func(t *testing.T) {
		data, err := GenerateQRPNG("EVVY-deadbeef1234")
		if err != nil {
			t.Fatalf("GenerateQRPNG() error = %v", err)
		}
		if len(data) == 0 {
			t.Fatal("expected non-empty PNG")
		}
		if _, err := png.Decode(bytes.NewReader(data)); err != nil {
			t.Fatalf("invalid PNG: %v", err)
		}
	})

	t.Run("empty barcode", func(t *testing.T) {
		_, err := GenerateQRPNG("")
		if err == nil {
			t.Fatal("expected error for empty barcode")
		}
	})
}
