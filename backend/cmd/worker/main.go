package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nnieru/mini-evvy/internal/config"
	"github.com/nnieru/mini-evvy/internal/database"
	"github.com/nnieru/mini-evvy/internal/mailer"
	"github.com/nnieru/mini-evvy/internal/repository"
	"github.com/nnieru/mini-evvy/internal/service"
	"github.com/nnieru/mini-evvy/internal/worker"
)

const (
	pollInterval   = 2 * time.Second
	maxRetries     = 3
	unpaidHold     = 1 * time.Hour
	expireInterval = 1 * time.Minute
)

func main() {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	jobRepo := repository.NewJobRepo(pool)
	guestRepo := repository.NewGuestRepo(pool)
	seatRepo := repository.NewSeatRepo(pool)
	bookingRepo := repository.NewBookingRepo(pool)
	eventRepo := repository.NewEventRepo(pool)
	categoryRepo := repository.NewSeatCategoryRepo(pool)
	userRoleRepo := repository.NewUserRoleRepo()
	emailTemplateRepo := repository.NewEventEmailTemplateRepo(pool)

	jobService := service.NewJobService(pool, jobRepo)
	bookingService := service.NewBookingService(
		pool,
		bookingRepo,
		eventRepo,
		guestRepo,
		seatRepo,
		categoryRepo,
		userRoleRepo,
		jobService,
	)
	mailAPIKey, err := cfg.MailAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	mailerClient, err := mailer.New(cfg.MailProvider, mailAPIKey, cfg.EmailFrom)
	if err != nil {
		log.Fatal(err)
	}

	processor := worker.NewProcessor(
		pool,
		jobRepo,
		guestRepo,
		seatRepo,
		bookingRepo,
		eventRepo,
		repository.NewSeatingDraftRepo(pool),
		emailTemplateRepo,
		jobService,
		mailerClient,
	)

	slog.Info("worker started")

	go runUnpaidExpiry(ctx, bookingService)

	for {
		if ctx.Err() != nil {
			slog.Info("worker shutting down")
			return
		}

		job, err := jobRepo.ClaimNext(ctx, pool)
		if errors.Is(err, repository.ErrNotFound) {
			time.Sleep(pollInterval)
			continue
		}
		if err != nil {
			slog.Error("claim job failed", "error", err)
			time.Sleep(pollInterval)
			continue
		}

		slog.Info("processing job", "job_id", job.ID, "type", job.Type)

		runErr := processor.Run(ctx, job)
		if runErr != nil {
			slog.Error("job failed", "job_id", job.ID, "type", job.Type, "error", runErr)
			retryCount := job.RetryCount + 1
			if err := jobRepo.MarkFailed(ctx, pool, job.ID, retryCount, maxRetries); err != nil {
				slog.Error("mark job failed", "job_id", job.ID, "error", err)
			}
			continue
		}

		if err := jobRepo.MarkDone(ctx, pool, job.ID); err != nil {
			slog.Error("mark job done", "job_id", job.ID, "error", err)
		}
	}
}

func runUnpaidExpiry(ctx context.Context, bookingService *service.BookingService) {
	ticker := time.NewTicker(expireInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-unpaidHold)
			n, err := bookingService.ExpireUnpaidBookings(ctx, cutoff)
			if err != nil {
				slog.Error("expire unpaid bookings failed", "error", err)
				continue
			}
			if n > 0 {
				slog.Info("expired unpaid bookings", "count", n)
			}
		}
	}
}
