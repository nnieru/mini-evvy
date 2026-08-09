package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/mailer/invitation"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
)

var ErrTargetNotEmpty = errors.New("target event already has seat categories or seats")

type ImportConfigInput struct {
	SourceEventID        uuid.UUID
	IncludeCategories    bool
	IncludeSeats         bool
	IncludeEmailTemplate bool
}

type ImportConfigResult struct {
	CategoriesCreated   int
	SeatsCreated        int
	EmailTemplateCopied bool
}

type importCategoryStore interface {
	Create(ctx context.Context, db repository.DBTX, c *model.SeatCategory) (*model.SeatCategory, error)
	ListByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID) ([]model.SeatCategory, error)
}

type importSeatStore interface {
	CreateBatch(ctx context.Context, db repository.DBTX, seats []model.Seat) ([]model.Seat, error)
	ListByEventID(ctx context.Context, db repository.DBTX, eventID uuid.UUID, status *model.SeatStatus, categoryID *uuid.UUID) ([]model.Seat, error)
}

type importEmailTemplateStore interface {
	GetByEventAndType(ctx context.Context, db repository.DBTX, eventID uuid.UUID, templateType string) (*model.EventEmailTemplate, error)
	Upsert(ctx context.Context, db repository.DBTX, t *model.EventEmailTemplate) (*model.EventEmailTemplate, error)
}

type EventImportService struct {
	pool        *pgxpool.Pool
	events      eventLookup
	categories  importCategoryStore
	seats       importSeatStore
	templates   importEmailTemplateStore
	memberships membershipChecker
}

func NewEventImportService(
	pool *pgxpool.Pool,
	events eventLookup,
	categories importCategoryStore,
	seats importSeatStore,
	templates importEmailTemplateStore,
	memberships membershipChecker,
) *EventImportService {
	return &EventImportService{
		pool:        pool,
		events:      events,
		categories:  categories,
		seats:       seats,
		templates:   templates,
		memberships: memberships,
	}
}

func ValidateImportConfigRequest(targetEventID, sourceEventID uuid.UUID, in ImportConfigInput) error {
	if sourceEventID == targetEventID {
		return fmt.Errorf("%w: source and target must be different events", ErrValidation)
	}
	if !in.IncludeCategories && !in.IncludeSeats && !in.IncludeEmailTemplate {
		return fmt.Errorf("%w: at least one include flag must be true", ErrValidation)
	}
	if in.IncludeSeats && !in.IncludeCategories {
		return fmt.Errorf("%w: include_categories is required when include_seats is true", ErrValidation)
	}
	return nil
}

func (s *EventImportService) requireStaff(ctx context.Context, actorID, eventID uuid.UUID) (*model.Event, error) {
	event, err := s.events.GetByID(ctx, s.pool, eventID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	can, err := s.memberships.HasRole(ctx, s.pool, actorID, event.OrganizationID, model.RoleOwner, model.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if !can {
		return nil, ErrForbidden
	}

	return event, nil
}

func (s *EventImportService) ImportConfig(
	ctx context.Context,
	actorID, targetEventID uuid.UUID,
	in ImportConfigInput,
) (*ImportConfigResult, error) {
	if err := ValidateImportConfigRequest(targetEventID, in.SourceEventID, in); err != nil {
		return nil, err
	}

	if _, err := s.requireStaff(ctx, actorID, targetEventID); err != nil {
		return nil, err
	}
	if _, err := s.requireStaff(ctx, actorID, in.SourceEventID); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	result := &ImportConfigResult{}

	if in.IncludeCategories || in.IncludeSeats {
		if err := s.ensureTargetSeatingEmpty(ctx, tx, targetEventID); err != nil {
			return nil, err
		}
	}

	categoryMap := make(map[uuid.UUID]uuid.UUID)

	if in.IncludeCategories {
		sourceCategories, err := s.categories.ListByEventID(ctx, tx, in.SourceEventID)
		if err != nil {
			return nil, fmt.Errorf("list source categories: %w", err)
		}
		for _, sourceCategory := range sourceCategories {
			created, err := s.categories.Create(ctx, tx, &model.SeatCategory{
				Name:     sourceCategory.Name,
				Code:     sourceCategory.Code,
				Price:    sourceCategory.Price,
				Currency: sourceCategory.Currency,
				EventID:  targetEventID,
			})
			if err != nil {
				return nil, fmt.Errorf("create category: %w", err)
			}
			categoryMap[sourceCategory.ID] = created.ID
			result.CategoriesCreated++
		}
	}

	if in.IncludeSeats {
		sourceSeats, err := s.seats.ListByEventID(ctx, tx, in.SourceEventID, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("list source seats: %w", err)
		}

		batch := make([]model.Seat, 0, len(sourceSeats))
		for _, sourceSeat := range sourceSeats {
			newCategoryID, ok := categoryMap[sourceSeat.CategoryID]
			if !ok {
				return nil, fmt.Errorf("%w: source seat %s references a category that was not copied", ErrValidation, sourceSeat.Code)
			}
			batch = append(batch, model.Seat{
				Code:        sourceSeat.Code,
				Description: sourceSeat.Description,
				Section:     sourceSeat.Section,
				Row:         sourceSeat.Row,
				Col:         sourceSeat.Col,
				Status:      model.SeatAvailable,
				EventID:     targetEventID,
				CategoryID:  newCategoryID,
			})
		}

		if len(batch) > 0 {
			if _, err := s.seats.CreateBatch(ctx, tx, batch); err != nil {
				return nil, fmt.Errorf("create seats: %w", err)
			}
		}
		result.SeatsCreated = len(batch)
	}

	if in.IncludeEmailTemplate {
		config, err := s.loadSourceInvitationConfig(ctx, tx, in.SourceEventID)
		if err != nil {
			return nil, err
		}
		if _, err := s.templates.Upsert(ctx, tx, &model.EventEmailTemplate{
			EventID:   targetEventID,
			Type:      model.EmailTemplateTypeInvitation,
			Config:    config,
			UpdatedBy: &actorID,
		}); err != nil {
			return nil, fmt.Errorf("upsert invitation template: %w", err)
		}
		result.EmailTemplateCopied = true
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}

func (s *EventImportService) ensureTargetSeatingEmpty(ctx context.Context, db repository.DBTX, targetEventID uuid.UUID) error {
	categories, err := s.categories.ListByEventID(ctx, db, targetEventID)
	if err != nil {
		return fmt.Errorf("list target categories: %w", err)
	}
	if len(categories) > 0 {
		return ErrTargetNotEmpty
	}

	seats, err := s.seats.ListByEventID(ctx, db, targetEventID, nil, nil)
	if err != nil {
		return fmt.Errorf("list target seats: %w", err)
	}
	if len(seats) > 0 {
		return ErrTargetNotEmpty
	}

	return nil
}

func (s *EventImportService) loadSourceInvitationConfig(ctx context.Context, db repository.DBTX, sourceEventID uuid.UUID) ([]byte, error) {
	row, err := s.templates.GetByEventAndType(ctx, db, sourceEventID, model.EmailTemplateTypeInvitation)
	if errors.Is(err, repository.ErrNotFound) {
		return invitation.DefaultConfig().Marshal()
	}
	if err != nil {
		return nil, fmt.Errorf("get source invitation template: %w", err)
	}
	return row.Config, nil
}
