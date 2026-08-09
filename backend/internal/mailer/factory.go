package mailer

import (
	"fmt"
	"strings"
)

func New(provider, apiKey, from string) (Mailer, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "plunk":
		if strings.TrimSpace(apiKey) == "" {
			return nil, fmt.Errorf("plunk api key not configured")
		}
		return NewPlunkMailer(apiKey, from), nil
	case "resend":
		if strings.TrimSpace(apiKey) == "" {
			return nil, fmt.Errorf("resend api key not configured")
		}
		return NewResendMailer(apiKey, from), nil
	default:
		return nil, fmt.Errorf("unknown mail provider: %s", provider)
	}
}
