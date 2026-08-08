package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobPending    JobStatus = "pending"
	JobProcessing JobStatus = "in_process"
	JobCompleted  JobStatus = "done"
	JobFailed     JobStatus = "failed"
)

type Job struct {
	ID         uuid.UUID
	Type       string
	Status     JobStatus
	Data       json.RawMessage
	RetryCount int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
