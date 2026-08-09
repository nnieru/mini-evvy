package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nnieru/mini-evvy/internal/config"
	"github.com/nnieru/mini-evvy/internal/database"
	"github.com/nnieru/mini-evvy/internal/handler"
	"github.com/nnieru/mini-evvy/internal/mailer"
	"github.com/nnieru/mini-evvy/internal/middleware"
	"github.com/nnieru/mini-evvy/internal/repository"
	"github.com/nnieru/mini-evvy/internal/service"
	"github.com/nnieru/mini-evvy/internal/storage"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)

	if err != nil {
		log.Fatal(err)
	}

	defer pool.Close()

	// roles seeder
	roleRepo := repository.NewRoleRepo(pool)
	if err := roleRepo.EnsureDefaults(ctx); err != nil {
		log.Fatal(err)
	}

	// users
	userRepo := repository.NewUserRepo(pool)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(authService)

	// organization
	orgRepo := repository.NewOrganizationRepo(pool)
	userRoleRepo := repository.NewUserRoleRepo()
	orgService := service.NewOrganizationService(pool, orgRepo, roleRepo, userRoleRepo, userRepo)
	orgHandler := handler.NewOrganizationHandler(orgService)

	// event
	eventRepo := repository.NewEventRepo(pool)
	eventService := service.NewEventService(pool, eventRepo, userRoleRepo)
	eventHandler := handler.NewEventHandler(eventService)

	// seat category
	categoryRepo := repository.NewSeatCategoryRepo(pool)
	categoryService := service.NewSeatCategoryService(pool, categoryRepo, eventRepo, userRoleRepo)
	categoryHandler := handler.NewSeatCategoryHandler(categoryService)

	// seats
	seatRepo := repository.NewSeatRepo(pool)
	seatService := service.NewSeatService(pool, seatRepo, eventRepo, categoryRepo, userRoleRepo)
	seatHandler := handler.NewSeatHandler(seatService)

	// guests
	guestRepo := repository.NewGuestRepo(pool)
	guestService := service.NewGuestService(pool, guestRepo, eventRepo, categoryRepo, userRoleRepo)
	guestHandler := handler.NewGuestHandler(guestService)

	// jobs
	jobRepo := repository.NewJobRepo(pool)
	jobService := service.NewJobService(pool, jobRepo)

	// bookings
	bookingRepo := repository.NewBookingRepo(pool)
	bookingService := service.NewBookingService(pool, bookingRepo, eventRepo, guestRepo, seatRepo, categoryRepo, userRoleRepo, jobService)

	// payments
	paymentRepo := repository.NewPaymentRepo(pool)
	paymentService := service.NewPaymentService(pool, paymentRepo, bookingRepo, eventRepo, seatRepo, guestRepo, userRoleRepo, jobService)
	paymentHandler := handler.NewPaymentHandler(paymentService)

	jobQueryService := service.NewJobQueryService(pool, jobRepo, bookingRepo, guestRepo, eventRepo, userRoleRepo, jobService)
	jobHandler := handler.NewJobHandler(jobQueryService)
	finalizeService := service.NewFinalizeService(pool, jobRepo, eventRepo, bookingRepo, seatRepo, userRoleRepo, jobService)
	eventJobsHandler := handler.NewEventJobsHandler(finalizeService)

	bookingHandler := handler.NewBookingHandler(bookingService, jobQueryService)

	// attendance
	attendanceRepo := repository.NewAttendanceRepo(pool)
	attendanceService := service.NewAttendanceService(pool, attendanceRepo, guestRepo, seatRepo, bookingRepo, eventRepo, userRoleRepo)
	attendanceHandler := handler.NewAttendanceHandler(attendanceService)

	emailTemplateRepo := repository.NewEventEmailTemplateRepo(pool)
	mailAPIKey, err := cfg.MailAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	mailerClient, err := mailer.New(cfg.MailProvider, mailAPIKey, cfg.EmailFrom)
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.ValidateS3(); err != nil {
		log.Fatal(err)
	}
	s3Client, err := storage.NewS3Client(
		cfg.S3Endpoint,
		cfg.S3Region,
		cfg.S3AccessKeyID,
		cfg.S3SecretAccessKey,
	)
	if err != nil {
		log.Fatal(err)
	}
	bannerStore := storage.NewBannerStore(s3Client, cfg.S3Bucket, cfg.S3PublicBaseURL)
	emailTemplateService := service.NewEmailTemplateService(
		pool,
		emailTemplateRepo,
		eventRepo,
		userRoleRepo,
		userRepo,
		mailerClient,
		bannerStore,
		cfg.S3PublicBaseURL,
	)
	emailTemplateHandler := handler.NewEmailTemplateHandler(emailTemplateService)
	eventImportService := service.NewEventImportService(
		pool,
		eventRepo,
		categoryRepo,
		seatRepo,
		emailTemplateRepo,
		userRoleRepo,
	)
	eventImportHandler := handler.NewEventImportHandler(eventImportService)

	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status": "ok", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)

	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg.JWTSecret))
		r.Get("/me", authHandler.Me)
		r.Get("/me/events", eventHandler.ListMine)

		r.Route("/orgs", func(r chi.Router) {
			r.Post("/", orgHandler.Create)
			r.Get("/", orgHandler.List)
			r.Get("/{orgId}", orgHandler.Get)
			r.Get("/{orgId}/my-role", orgHandler.GetMyRole)
			r.Post("/{orgId}/members", orgHandler.AddMember)
			r.Get("/{orgId}/members", orgHandler.ListMembers)

			r.Post("/{orgId}/events", eventHandler.Create)
			r.Get("/{orgId}/events", eventHandler.List)
		})

		r.Get("/events/{eventId}", eventHandler.Get)
		r.Patch("/events/{eventId}", eventHandler.Update)
		r.Post("/events/{eventId}/import-config", eventImportHandler.ImportConfig)
		r.Post("/events/{eventId}/finalize-seating", eventJobsHandler.FinalizeSeating)
		r.Get("/events/{eventId}/seating-preview", eventJobsHandler.GetSeatingPreview)
		r.Post("/events/{eventId}/seating-approve", eventJobsHandler.ApproveSeating)
		r.Post("/events/{eventId}/seating-reject", eventJobsHandler.RejectSeating)
		r.Get("/events/{eventId}/jobs", jobHandler.ListByEvent)

		r.Get("/events/{eventId}/email-template/invitation", emailTemplateHandler.GetInvitation)
		r.Put("/events/{eventId}/email-template/invitation", emailTemplateHandler.UpsertInvitation)
		r.Delete("/events/{eventId}/email-template/invitation", emailTemplateHandler.ResetInvitation)
		r.Post("/events/{eventId}/email-template/invitation/preview", emailTemplateHandler.PreviewInvitation)
		r.Post("/events/{eventId}/email-template/invitation/test-send", emailTemplateHandler.TestSendInvitation)
		r.Post("/events/{eventId}/email-template/invitation/banner", emailTemplateHandler.UploadBanner)

		r.Get("/jobs/{jobId}", jobHandler.Get)

		r.Post("/events/{eventId}/categories", categoryHandler.Create)
		r.Get("/events/{eventId}/categories", categoryHandler.List)
		r.Get("/categories/{categoryId}", categoryHandler.Get)
		r.Patch("/categories/{categoryId}", categoryHandler.Update)

		r.Post("/events/{eventId}/seats", seatHandler.Create)
		r.Get("/events/{eventId}/seats", seatHandler.List)
		r.Get("/seats/{seatId}", seatHandler.Get)
		r.Patch("/seats/{seatId}", seatHandler.Update)

		r.Post("/events/{eventId}/guests", guestHandler.Create)
		r.Post("/events/{eventId}/guests/import", guestHandler.Import)
		r.Get("/events/{eventId}/guests", guestHandler.List)
		r.Get("/guests/{guestId}", guestHandler.Get)
		r.Patch("/guests/{guestId}", guestHandler.Update)

		r.Post("/events/{eventId}/bookings", bookingHandler.Create)
		r.Post("/events/{eventId}/bookings/batch", bookingHandler.CreateBatch)
		r.Get("/events/{eventId}/bookings", bookingHandler.List)
		r.Get("/bookings/{bookingId}", bookingHandler.Get)
		r.Patch("/bookings/{bookingId}", bookingHandler.Update)
		r.Delete("/bookings/{bookingId}", bookingHandler.Delete)
		r.Post("/bookings/{bookingId}/resend-invitation", bookingHandler.ResendInvitation)

		r.Post("/bookings/{bookingId}/payments", paymentHandler.Create)
		r.Get("/bookings/{bookingId}/payments", paymentHandler.List)

		r.Post("/events/{eventId}/attendance", attendanceHandler.Create)
		r.Get("/events/{eventId}/attendance", attendanceHandler.List)
		r.Get("/attendance/{attendanceId}", attendanceHandler.Get)
		r.Patch("/attendance/{attendanceId}", attendanceHandler.Update)
		r.Delete("/attendance/{attendanceId}", attendanceHandler.Delete)
	})

	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Println("listening on", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-shutdownCtx.Done()
	log.Println("shutting down")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(timeoutCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
