package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/mailer"
	"github.com/nnieru/mini-evvy/internal/mailer/invitation"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
	"github.com/nnieru/mini-evvy/internal/storage"
)

var (
	ErrEmailTemplateNotFound = errors.New("email template not found")
	ErrTestSendRateLimited   = errors.New("test send rate limited; try again in a minute")
	ErrValidation            = errors.New("validation error")
)

type invitationTemplateLoader interface {
	GetByEventAndType(ctx context.Context, db repository.DBTX, eventID uuid.UUID, templateType string) (*model.EventEmailTemplate, error)
}

type emailTemplateStore interface {
	invitationTemplateLoader
	Upsert(ctx context.Context, db repository.DBTX, t *model.EventEmailTemplate) (*model.EventEmailTemplate, error)
	DeleteByEventAndType(ctx context.Context, db repository.DBTX, eventID uuid.UUID, templateType string) error
}

type emailTemplateUserLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type bannerUploader interface {
	SaveEventBanner(eventID uuid.UUID, originalName string, content []byte) (string, error)
	PublicBaseURL() string
}

type EmailTemplateService struct {
	pool          *pgxpool.Pool
	templates     emailTemplateStore
	events        eventLookup
	memberships   membershipChecker
	users         emailTemplateUserLookup
	mailer        mailer.Mailer
	banners       bannerUploader
	publicBaseURL string

	testSendMu   sync.Mutex
	lastTestSend map[uuid.UUID]time.Time
}

func NewEmailTemplateService(
	pool *pgxpool.Pool,
	templates emailTemplateStore,
	events eventLookup,
	memberships membershipChecker,
	users emailTemplateUserLookup,
	mailerClient mailer.Mailer,
	banners bannerUploader,
	publicBaseURL string,
) *EmailTemplateService {
	return &EmailTemplateService{
		pool:          pool,
		templates:     templates,
		events:        events,
		memberships:   memberships,
		users:         users,
		mailer:        mailerClient,
		banners:       banners,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),

		lastTestSend: make(map[uuid.UUID]time.Time),
	}
}

type EmailTemplateView struct {
	Config    invitation.Config
	IsDefault bool
	UpdatedAt *time.Time
}

type PreviewResult struct {
	Subject string
	HTML    string
}

func (s *EmailTemplateService) requireStaff(ctx context.Context, actorID, eventID uuid.UUID) (*model.Event, error) {
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

func (s *EmailTemplateService) GetInvitation(ctx context.Context, actorID, eventID uuid.UUID) (*EmailTemplateView, error) {
	if _, err := s.requireStaff(ctx, actorID, eventID); err != nil {
		return nil, err
	}

	row, err := s.templates.GetByEventAndType(ctx, s.pool, eventID, model.EmailTemplateTypeInvitation)
	if errors.Is(err, repository.ErrNotFound) {
		return &EmailTemplateView{
			Config:    invitation.DefaultConfig(),
			IsDefault: true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get invitation template: %w", err)
	}

	cfg, err := invitation.ParseConfig(row.Config)
	if err != nil {
		return nil, err
	}

	return &EmailTemplateView{
		Config:    cfg,
		IsDefault: false,
		UpdatedAt: &row.UpdatedAt,
	}, nil
}

func (s *EmailTemplateService) UpsertInvitation(ctx context.Context, actorID, eventID uuid.UUID, cfg invitation.Config) (*EmailTemplateView, error) {
	if _, err := s.requireStaff(ctx, actorID, eventID); err != nil {
		return nil, err
	}

	cfg = invitation.NormalizeConfig(cfg.Sanitized())
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := s.validateBannerURL(eventID, cfg.BannerImageURL); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	raw, err := cfg.Marshal()
	if err != nil {
		return nil, err
	}

	row, err := s.templates.Upsert(ctx, s.pool, &model.EventEmailTemplate{
		EventID:   eventID,
		Type:      model.EmailTemplateTypeInvitation,
		Config:    raw,
		UpdatedBy: &actorID,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert invitation template: %w", err)
	}

	parsed, err := invitation.ParseConfig(row.Config)
	if err != nil {
		return nil, err
	}

	return &EmailTemplateView{
		Config:    parsed,
		IsDefault: false,
		UpdatedAt: &row.UpdatedAt,
	}, nil
}

func (s *EmailTemplateService) ResetInvitation(ctx context.Context, actorID, eventID uuid.UUID) error {
	if _, err := s.requireStaff(ctx, actorID, eventID); err != nil {
		return err
	}

	err := s.templates.DeleteByEventAndType(ctx, s.pool, eventID, model.EmailTemplateTypeInvitation)
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	return err
}

func (s *EmailTemplateService) PreviewInvitation(ctx context.Context, actorID, eventID uuid.UUID, cfg *invitation.Config) (*PreviewResult, error) {
	if _, err := s.requireStaff(ctx, actorID, eventID); err != nil {
		return nil, err
	}

	active := invitation.DefaultConfig()
	if cfg != nil {
		active = invitation.NormalizeConfig(cfg.Sanitized())
		if err := active.Validate(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidation, err)
		}
		if err := s.validateBannerURL(eventID, active.BannerImageURL); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidation, err)
		}
	} else {
		view, err := s.GetInvitation(ctx, actorID, eventID)
		if err != nil {
			return nil, err
		}
		active = view.Config
	}

	rendered, err := invitation.Render(active, invitation.SampleContext(), true)
	if err != nil {
		return nil, fmt.Errorf("render preview: %w", err)
	}

	return &PreviewResult{
		Subject: rendered.Subject,
		HTML:    rendered.HTML,
	}, nil
}

func (s *EmailTemplateService) TestSendInvitation(ctx context.Context, actorID, eventID uuid.UUID, cfg *invitation.Config) error {
	if _, err := s.requireStaff(ctx, actorID, eventID); err != nil {
		return err
	}

	if err := s.checkTestSendRateLimit(actorID); err != nil {
		return err
	}

	user, err := s.users.GetByID(ctx, actorID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if user.Email == "" {
		return fmt.Errorf("%w: user has no email", ErrValidation)
	}

	active := invitation.DefaultConfig()
	if cfg != nil {
		active = invitation.NormalizeConfig(cfg.Sanitized())
		if err := active.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
		if err := s.validateBannerURL(eventID, active.BannerImageURL); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
	} else {
		view, err := s.GetInvitation(ctx, actorID, eventID)
		if err != nil {
			return err
		}
		active = view.Config
	}

	sample := invitation.SampleContext()
	sample.TicketCode = fmt.Sprintf("EVVY-test-%s", uuid.New().String()[:8])
	sample.UniqueSendID = uuid.New().String()

	rendered, err := invitation.Render(active, sample, false)
	if err != nil {
		return fmt.Errorf("render test email: %w", err)
	}

	if s.mailer == nil {
		return fmt.Errorf("mailer not configured")
	}

	subject := fmt.Sprintf("[Test] %s · %s", rendered.Subject, time.Now().Format("2006-01-02 15:04:05"))
	if err := s.mailer.Send(ctx, user.Email, subject, rendered.Text, rendered.HTML, rendered.Attachments...); err != nil {
		slog.Error("resend test send failed", "to", user.Email, "error", err)
		return fmt.Errorf("send test email: %w", err)
	}

	return nil
}

func (s *EmailTemplateService) checkTestSendRateLimit(actorID uuid.UUID) error {
	s.testSendMu.Lock()
	defer s.testSendMu.Unlock()

	now := time.Now()
	if last, ok := s.lastTestSend[actorID]; ok && now.Sub(last) < time.Minute {
		return ErrTestSendRateLimited
	}
	s.lastTestSend[actorID] = now
	return nil
}

func LoadInvitationConfig(ctx context.Context, db repository.DBTX, store invitationTemplateLoader, eventID uuid.UUID) (invitation.Config, error) {
	row, err := store.GetByEventAndType(ctx, db, eventID, model.EmailTemplateTypeInvitation)
	if errors.Is(err, repository.ErrNotFound) {
		return invitation.DefaultConfig(), nil
	}
	if err != nil {
		return invitation.Config{}, fmt.Errorf("get invitation template: %w", err)
	}
	return invitation.ParseConfig(row.Config)
}

func (s *EmailTemplateService) UploadBanner(
	ctx context.Context,
	actorID, eventID uuid.UUID,
	filename string,
	content []byte,
) (string, error) {
	if _, err := s.requireStaff(ctx, actorID, eventID); err != nil {
		return "", err
	}
	if s.banners == nil {
		return "", fmt.Errorf("banner uploads not configured")
	}
	return s.banners.SaveEventBanner(eventID, filename, content)
}

func (s *EmailTemplateService) validateBannerURL(eventID uuid.UUID, raw *string) error {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}

	value := strings.TrimSpace(*raw)
	if storage.IsUploadedBannerURL(value, eventID, s.publicBaseURL) {
		return nil
	}

	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("banner_image_url must be https or an uploaded banner for this event")
	}
	return nil
}
