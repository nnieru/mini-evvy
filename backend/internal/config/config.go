package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTSecret        string
	MailProvider     string
	PlunkAPIKey      string
	ResendAPIKey     string
	EmailFrom        string
	S3Endpoint       string
	S3Region         string
	S3AccessKeyID    string
	S3SecretAccessKey string
	S3Bucket         string
	S3PublicBaseURL  string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found")
	}

	return Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		JWTSecret:         getEnv("JWT_SECRET", "some-long-random-dev-string"),
		MailProvider:      getEnv("MAIL_PROVIDER", "plunk"),
		PlunkAPIKey:       getEnv("PLUNK_API_KEY", ""),
		ResendAPIKey:      getEnv("RESEND_API_KEY", ""),
		EmailFrom:         getEnv("EMAIL_FROM", "onboarding@resend.dev"),
		S3Endpoint:        getEnv("S3_ENDPOINT", ""),
		S3Region:          getEnv("S3_REGION", ""),
		S3AccessKeyID:     getEnv("S3_ACCESS_KEY_ID", ""),
		S3SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
		S3Bucket:          getEnv("S3_BUCKET", ""),
		S3PublicBaseURL:   getEnv("S3_PUBLIC_BASE_URL", ""),
	}
}

func (c Config) MailAPIKey() (string, error) {
	switch strings.ToLower(strings.TrimSpace(c.MailProvider)) {
	case "plunk":
		if strings.TrimSpace(c.PlunkAPIKey) == "" {
			return "", fmt.Errorf("PLUNK_API_KEY required when MAIL_PROVIDER=plunk")
		}
		return c.PlunkAPIKey, nil
	case "resend":
		if strings.TrimSpace(c.ResendAPIKey) == "" {
			return "", fmt.Errorf("RESEND_API_KEY required when MAIL_PROVIDER=resend")
		}
		return c.ResendAPIKey, nil
	default:
		return "", fmt.Errorf("unknown MAIL_PROVIDER: %s", c.MailProvider)
	}
}

func (c Config) ValidateS3() error {
	missing := make([]string, 0, 6)
	if strings.TrimSpace(c.S3Endpoint) == "" {
		missing = append(missing, "S3_ENDPOINT")
	}
	if strings.TrimSpace(c.S3Region) == "" {
		missing = append(missing, "S3_REGION")
	}
	if strings.TrimSpace(c.S3AccessKeyID) == "" {
		missing = append(missing, "S3_ACCESS_KEY_ID")
	}
	if strings.TrimSpace(c.S3SecretAccessKey) == "" {
		missing = append(missing, "S3_SECRET_ACCESS_KEY")
	}
	if strings.TrimSpace(c.S3Bucket) == "" {
		missing = append(missing, "S3_BUCKET")
	}
	if strings.TrimSpace(c.S3PublicBaseURL) == "" {
		missing = append(missing, "S3_PUBLIC_BASE_URL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing S3 config: %s", strings.Join(missing, ", "))
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}
