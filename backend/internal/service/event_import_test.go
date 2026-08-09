package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestValidateImportConfigRequest(t *testing.T) {
	targetID := uuid.New()
	sourceID := uuid.New()

	t.Run("same event rejected", func(t *testing.T) {
		err := ValidateImportConfigRequest(targetID, targetID, ImportConfigInput{
			IncludeCategories: true,
		})
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("no includes rejected", func(t *testing.T) {
		err := ValidateImportConfigRequest(targetID, sourceID, ImportConfigInput{})
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("seats without categories rejected", func(t *testing.T) {
		err := ValidateImportConfigRequest(targetID, sourceID, ImportConfigInput{
			IncludeSeats: true,
		})
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("email only ok", func(t *testing.T) {
		if err := ValidateImportConfigRequest(targetID, sourceID, ImportConfigInput{
			IncludeEmailTemplate: true,
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("categories and seats ok", func(t *testing.T) {
		if err := ValidateImportConfigRequest(targetID, sourceID, ImportConfigInput{
			IncludeCategories: true,
			IncludeSeats:      true,
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
