package storage

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateBannerImagePNG(t *testing.T) {
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	}

	contentType, ext, err := validateBannerImage(png, "banner.png")
	if err != nil {
		t.Fatalf("validateBannerImage() error = %v", err)
	}
	if contentType != "image/png" {
		t.Fatalf("unexpected content type %q", contentType)
	}
	if ext != ".png" {
		t.Fatalf("unexpected ext %q", ext)
	}
}

func TestValidateBannerImageRejectsNonImage(t *testing.T) {
	_, _, err := validateBannerImage([]byte("hello"), "notes.txt")
	if err == nil {
		t.Fatal("expected error for non-image upload")
	}
}

func TestIsUploadedBannerURL(t *testing.T) {
	eventID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	base := "https://example.supabase.co/storage/v1/object/public/email-banners"
	url := base + "/events/" + eventID.String() + "/abc.png"

	if !IsUploadedBannerURL(url, eventID, base) {
		t.Fatalf("expected uploaded banner url to match")
	}
	if IsUploadedBannerURL("https://evil.com/events/"+eventID.String()+"/x.png", eventID, base) {
		t.Fatal("expected foreign url to not match")
	}
}
