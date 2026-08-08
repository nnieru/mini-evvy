package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
)

var (
	ErrFinalizeInProgress = errors.New("finalize seating already in progress")
)

type jobStore interface {
	Create(ctx context.Context, db repository.DBTX, job *model.Job) (*model.Job, error)
	ExistsFinalizeInProgress(ctx context.Context, db repository.DBTX, eventID uuid.UUID) (bool, error)
}

type JobService struct {
	pool *pgxpool.Pool
	jobs jobStore
}

func NewJobService(pool *pgxpool.Pool, jobs jobStore) *JobService {
	return &JobService{pool: pool, jobs: jobs}
}

func (s *JobService) Enqueue(ctx context.Context, jobType string, data any) (*model.Job, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal job data: %w", err)
	}

	job := &model.Job{
		Type:       jobType,
		Status:     model.JobPending,
		RetryCount: 0,
		Data:       raw,
	}

	created, err := s.jobs.Create(ctx, s.pool, job)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	return created, nil
}

func (s *JobService) EnqueueTx(ctx context.Context, db repository.DBTX, jobType string, data any) (*model.Job, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal job data: %w", err)
	}

	job := &model.Job{
		Type:       jobType,
		Status:     model.JobPending,
		RetryCount: 0,
		Data:       raw,
	}

	created, err := s.jobs.Create(ctx, db, job)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	return created, nil
}
