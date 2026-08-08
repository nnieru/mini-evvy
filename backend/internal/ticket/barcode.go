package ticket

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const barcodePrefix = "EVVY-"

func GenerateBarcode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return barcodePrefix + hex.EncodeToString(b), nil
}
