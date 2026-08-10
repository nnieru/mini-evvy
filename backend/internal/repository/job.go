package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/jobtype"
	"github.com/nnieru/mini-evvy/internal/model"
)

type JobRepo struct {
	pool *pgxpool.Pool
}

func NewJobRepo(pool *pgxpool.Pool) *JobRepo {
	return &JobRepo{pool: pool}
}

func (r *JobRepo) Create(ctx context.Context, db DBTX, job *model.Job) (*model.Job, error) {
	const query = `
		INSERT INTO jobs (type, status, retry_count, data)
		VALUES ($1, $2, $3, $4)
		RETURNING id, type, status, retry_count, data, created_at, updated_at
	`

	var out model.Job
	err := db.QueryRow(ctx, query,
		job.Type,
		job.Status,
		job.RetryCount,
		job.Data,
	).Scan(
		&out.ID,
		&out.Type,
		&out.Status,
		&out.RetryCount,
		&out.Data,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	return &out, nil
}

func (r *JobRepo) ClaimNext(ctx context.Context, db DBTX) (*model.Job, error) {
	const query = `
		UPDATE jobs SET status = 'in_process', updated_at = now()
		WHERE id = (
			SELECT id FROM jobs
			WHERE status = 'pending'
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, type, status, retry_count, data, created_at, updated_at
	`

	var job model.Job
	err := db.QueryRow(ctx, query).Scan(
		&job.ID,
		&job.Type,
		&job.Status,
		&job.RetryCount,
		&job.Data,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("claim next job: %w", err)
	}

	return &job, nil
}

func (r *JobRepo) MarkDone(ctx context.Context, db DBTX, id uuid.UUID) error {
	const query = `
		UPDATE jobs SET status = 'done', updated_at = now()
		WHERE id = $1
	`

	tag, err := db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark job done: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *JobRepo) MarkFailed(ctx context.Context, db DBTX, id uuid.UUID, retryCount int, maxRetries int) error {
	var query string
	var status model.JobStatus
	if retryCount >= maxRetries {
		status = model.JobFailed
		query = `UPDATE jobs SET status = $1, retry_count = $2, updated_at = now() WHERE id = $3`
	} else {
		status = model.JobPending
		query = `UPDATE jobs SET status = $1, retry_count = $2, updated_at = now() WHERE id = $3`
	}

	tag, err := db.Exec(ctx, query, status, retryCount, id)
	if err != nil {
		return fmt.Errorf("mark job failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *JobRepo) ExistsFinalizeInProgress(ctx context.Context, db DBTX, eventID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM jobs
			WHERE type = $1
				AND status IN ('pending', 'in_process')
				AND data->>'event_id' = $2
		)
	`

	var exists bool
	err := db.QueryRow(ctx, query, jobtype.FinalizeSeating, eventID.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("exists finalize in progress: %w", err)
	}
	return exists, nil
}

func (r *JobRepo) GetByID(ctx context.Context, db DBTX, id uuid.UUID) (*model.Job, error) {
	const query = `
		SELECT id, type, status, retry_count, data, created_at, updated_at
		FROM jobs
		WHERE id = $1
	`

	var job model.Job
	err := db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&job.Type,
		&job.Status,
		&job.RetryCount,
		&job.Data,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job by id: %w", err)
	}

	return &job, nil
}

type JobListQuery struct {
	Page     int
	PageSize int
	Q        string
	Type     string
	Status   string
}

func jobListFilterClause(q JobListQuery, startPos int) (string, []any, int) {
	clause := ""
	args := []any{}
	pos := startPos

	if strings.TrimSpace(q.Type) != "" {
		clause += fmt.Sprintf(" AND type = $%d", pos)
		args = append(args, q.Type)
		pos++
	}
	if strings.TrimSpace(q.Status) != "" {
		clause += fmt.Sprintf(" AND status = $%d", pos)
		args = append(args, q.Status)
		pos++
	}
	if strings.TrimSpace(q.Q) != "" {
		clause += fmt.Sprintf(" AND (type::text ILIKE $%d OR status::text ILIKE $%d)", pos, pos)
		args = append(args, SearchPattern(q.Q))
		pos++
	}

	return clause, args, pos
}

func invitationReportSearchClause(q JobListQuery, startPos int) (string, []any, int) {
	if strings.TrimSpace(q.Q) == "" {
		return "", nil, startPos
	}
	pattern := SearchPattern(q.Q)
	clause := fmt.Sprintf(
		` AND (
			COALESCE(g.name, '') ILIKE $%d OR
			COALESCE(g.email, '') ILIKE $%d OR
			COALESCE(s.code, '') ILIKE $%d OR
			COALESCE(b.barcode, '') ILIKE $%d OR
			j.status::text ILIKE $%d
		)`,
		startPos, startPos, startPos, startPos, startPos,
	)
	return clause, []any{pattern}, startPos + 1
}

func (r *JobRepo) CountByEventIDFiltered(ctx context.Context, db DBTX, eventID uuid.UUID, q JobListQuery) (int, error) {
	args := []any{eventID.String()}
	query := `SELECT COUNT(*) FROM jobs j WHERE j.data->>'event_id' = $1`
	filterClause, filterArgs, _ := jobListFilterClause(q, 2)
	query += filterClause
	args = append(args, filterArgs...)

	var total int
	if err := db.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count jobs by event id: %w", err)
	}
	return total, nil
}

func (r *JobRepo) ListByEventIDPaged(ctx context.Context, db DBTX, eventID uuid.UUID, q JobListQuery) ([]model.Job, error) {
	args := []any{eventID.String()}
	query := `
		SELECT id, type, status, retry_count, data, created_at, updated_at
		FROM jobs
		WHERE data->>'event_id' = $1`
	filterClause, filterArgs, nextPos := jobListFilterClause(q, 2)
	query += filterClause
	args = append(args, filterArgs...)

	offset := (q.Page - 1) * q.PageSize
	limitPos := nextPos
	offsetPos := nextPos + 1
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, limitPos, offsetPos)
	args = append(args, q.PageSize, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs by event id paged: %w", err)
	}
	defer rows.Close()

	var out []model.Job
	for rows.Next() {
		var job model.Job
		if err := rows.Scan(
			&job.ID,
			&job.Type,
			&job.Status,
			&job.RetryCount,
			&job.Data,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		out = append(out, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list jobs paged rows: %w", err)
	}

	return out, nil
}

func (r *JobRepo) ListByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) ([]model.Job, error) {
	const query = `
		SELECT id, type, status, retry_count, data, created_at, updated_at
		FROM jobs
		WHERE data->>'event_id' = $1
		ORDER BY created_at DESC
	`

	rows, err := db.Query(ctx, query, eventID.String())
	if err != nil {
		return nil, fmt.Errorf("list jobs by event id: %w", err)
	}
	defer rows.Close()

	var out []model.Job
	for rows.Next() {
		var job model.Job
		if err := rows.Scan(
			&job.ID,
			&job.Type,
			&job.Status,
			&job.RetryCount,
			&job.Data,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		out = append(out, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list jobs rows: %w", err)
	}

	return out, nil
}

func (r *JobRepo) ListInvitationReportByEventID(
	ctx context.Context,
	db DBTX,
	eventID uuid.UUID,
	q JobListQuery,
) ([]model.InvitationEmailReportRow, error) {
	args := []any{eventID.String(), jobtype.SendInvitation}
	query := `
		SELECT
			COALESCE(g.name, ''),
			COALESCE(sc.name, ''),
			COALESCE(s.code, ''),
			COALESCE(g.email, ''),
			COALESCE(b.barcode, ''),
			j.status
		FROM jobs j
		LEFT JOIN seat_bookings b ON b.id = (j.data->>'booking_id')::uuid AND b.deleted_at IS NULL
		LEFT JOIN guests g ON g.id = b.guest_id AND g.deleted_at IS NULL
		LEFT JOIN seats s ON s.id = b.seat_id AND s.deleted_at IS NULL
		LEFT JOIN seat_categories sc ON sc.id = b.category_id AND sc.deleted_at IS NULL
		WHERE j.data->>'event_id' = $1
			AND j.type = $2`

	nextPos := 3
	if strings.TrimSpace(q.Status) != "" {
		query += fmt.Sprintf(" AND j.status = $%d", nextPos)
		args = append(args, q.Status)
		nextPos++
	}
	searchClause, searchArgs, _ := invitationReportSearchClause(q, nextPos)
	query += searchClause
	args = append(args, searchArgs...)
	query += ` ORDER BY j.created_at DESC`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list invitation report by event id: %w", err)
	}
	defer rows.Close()

	var out []model.InvitationEmailReportRow
	for rows.Next() {
		var row model.InvitationEmailReportRow
		if err := rows.Scan(
			&row.GuestName,
			&row.CategoryName,
			&row.SeatCode,
			&row.GuestEmail,
			&row.TicketCode,
			&row.JobStatus,
		); err != nil {
			return nil, fmt.Errorf("scan invitation report row: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list invitation report rows: %w", err)
	}
	return out, nil
}

func (r *JobRepo) ReclaimInProcessJobs(ctx context.Context, db DBTX) (int64, error) {
	const query = `
		UPDATE jobs
		SET status = 'pending', updated_at = now()
		WHERE status = 'in_process'
	`

	tag, err := db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("reclaim in process jobs: %w", err)
	}
	return tag.RowsAffected(), nil
}
