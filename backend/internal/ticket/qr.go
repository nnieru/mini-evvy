package ticket

import (
	"fmt"

	"github.com/skip2/go-qrcode"
)

const qrSize = 256

func GenerateQRPNG(barcode string) ([]byte, error) {
	if barcode == "" {
		return nil, fmt.Errorf("barcode required")
	}

	png, err := qrcode.Encode(barcode, qrcode.Medium, qrSize)
	if err != nil {
		return nil, fmt.Errorf("encode qr code: %w", err)
	}

	return png, nil
}
