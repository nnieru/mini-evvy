package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type BannerStore struct {
	client          *s3.Client
	bucket          string
	publicBaseURL   string
}

func NewBannerStore(client *s3.Client, bucket, publicBaseURL string) *BannerStore {
	return &BannerStore{
		client:        client,
		bucket:        bucket,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}
}

const maxBannerBytes = 2 << 20 // 2 MiB

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

func (s *BannerStore) SaveEventBanner(eventID uuid.UUID, originalName string, content []byte) (string, error) {
	contentType, ext, err := validateBannerImage(content, originalName)
	if err != nil {
		return "", err
	}

	name := uuid.New().String() + ext
	key := fmt.Sprintf("events/%s/%s", eventID.String(), name)

	_, err = s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("upload banner to s3: %w", err)
	}

	publicURL := fmt.Sprintf("%s/%s", s.publicBaseURL, key)
	return publicURL, nil
}

func (s *BannerStore) PublicBaseURL() string {
	return s.publicBaseURL
}

func validateBannerImage(content []byte, originalName string) (contentType, ext string, err error) {
	if len(content) == 0 {
		return "", "", fmt.Errorf("empty file")
	}
	if len(content) > maxBannerBytes {
		return "", "", fmt.Errorf("file too large (max 2 MiB)")
	}

	contentType = http.DetectContentType(content)
	ext, ok := allowedImageTypes[contentType]
	if !ok {
		return "", "", fmt.Errorf("unsupported image type (use JPEG, PNG, WebP, or GIF)")
	}

	if ext == ".jpg" && strings.HasSuffix(strings.ToLower(originalName), ".jpeg") {
		ext = ".jpeg"
	}

	return contentType, ext, nil
}

func IsUploadedBannerURL(raw string, eventID uuid.UUID, publicBaseURL string) bool {
	prefix := strings.TrimRight(publicBaseURL, "/") + "/events/" + eventID.String() + "/"
	return strings.HasPrefix(strings.TrimSpace(raw), prefix)
}
