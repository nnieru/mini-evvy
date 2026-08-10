package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
)

type SeatingDraftRepo struct {
	pool *pgxpool.Pool
}

func NewSeatingDraftRepo(pool *pgxpool.Pool) *SeatingDraftRepo {
	return &SeatingDraftRepo{pool: pool}
}

func (r *SeatingDraftRepo) Create(ctx context.Context, db DBTX, draft *model.SeatingDraft) (*model.SeatingDraft, error) {
	const query = `
		INSERT INTO seating_drafts (event_id, status, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, event_id, status, created_by, created_at, updated_at
	`

	var out model.SeatingDraft
	err := db.QueryRow(ctx, query, draft.EventID, draft.Status, draft.CreatedBy).Scan(
		&out.ID,
		&out.EventID,
		&out.Status,
		&out.CreatedBy,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create seating draft: %w", err)
	}
	return &out, nil
}

func (r *SeatingDraftRepo) GetOpenByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) (*model.SeatingDraft, error) {
	const query = `
		SELECT id, event_id, status, created_by, created_at, updated_at
		FROM seating_drafts
		WHERE event_id = $1 AND status = 'open'
		LIMIT 1
	`

	var draft model.SeatingDraft
	err := db.QueryRow(ctx, query, eventID).Scan(
		&draft.ID,
		&draft.EventID,
		&draft.Status,
		&draft.CreatedBy,
		&draft.CreatedAt,
		&draft.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get open seating draft: %w", err)
	}
	return &draft, nil
}

func (r *SeatingDraftRepo) HasApprovedByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM seating_drafts
			WHERE event_id = $1 AND status = 'approved'
		)
	`
	var exists bool
	if err := db.QueryRow(ctx, query, eventID).Scan(&exists); err != nil {
		return false, fmt.Errorf("has approved seating draft: %w", err)
	}
	return exists, nil
}

func (r *SeatingDraftRepo) UpdateStatus(ctx context.Context, db DBTX, draftID uuid.UUID, status model.SeatingDraftStatus) error {
	const query = `
		UPDATE seating_drafts SET status = $1, updated_at = now()
		WHERE id = $2
	`
	tag, err := db.Exec(ctx, query, status, draftID)
	if err != nil {
		return fmt.Errorf("update seating draft status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SeatingDraftRepo) CreateItem(ctx context.Context, db DBTX, item *model.SeatingDraftItem) (*model.SeatingDraftItem, error) {
	const query = `
		INSERT INTO seating_draft_items (draft_id, guest_id, seat_id, category_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, draft_id, guest_id, seat_id, category_id, created_at
	`

	var out model.SeatingDraftItem
	err := db.QueryRow(ctx, query, item.DraftID, item.GuestID, item.SeatID, item.CategoryID).Scan(
		&out.ID,
		&out.DraftID,
		&out.GuestID,
		&out.SeatID,
		&out.CategoryID,
		&out.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create seating draft item: %w", err)
	}
	return &out, nil
}

func (r *SeatingDraftRepo) GetItemByID(ctx context.Context, db DBTX, itemID uuid.UUID) (*model.SeatingDraftItem, error) {
	const query = `
		SELECT id, draft_id, guest_id, seat_id, category_id, created_at
		FROM seating_draft_items
		WHERE id = $1
	`

	var item model.SeatingDraftItem
	err := db.QueryRow(ctx, query, itemID).Scan(
		&item.ID,
		&item.DraftID,
		&item.GuestID,
		&item.SeatID,
		&item.CategoryID,
		&item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get seating draft item: %w", err)
	}
	return &item, nil
}

func (r *SeatingDraftRepo) UpdateItemSeat(ctx context.Context, db DBTX, itemID, seatID, categoryID uuid.UUID) error {
	const query = `
		UPDATE seating_draft_items
		SET seat_id = $1, category_id = $2
		WHERE id = $3
	`
	tag, err := db.Exec(ctx, query, seatID, categoryID, itemID)
	if err != nil {
		return fmt.Errorf("update seating draft item seat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SeatingDraftRepo) ListItemsByDraftID(ctx context.Context, db DBTX, draftID uuid.UUID) ([]model.SeatingDraftItem, error) {
	const query = `
		SELECT id, draft_id, guest_id, seat_id, category_id, created_at
		FROM seating_draft_items
		WHERE draft_id = $1
		ORDER BY created_at ASC
	`

	rows, err := db.Query(ctx, query, draftID)
	if err != nil {
		return nil, fmt.Errorf("list seating draft items: %w", err)
	}
	defer rows.Close()

	var out []model.SeatingDraftItem
	for rows.Next() {
		var item model.SeatingDraftItem
		if err := rows.Scan(
			&item.ID,
			&item.DraftID,
			&item.GuestID,
			&item.SeatID,
			&item.CategoryID,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan seating draft item: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list seating draft items rows: %w", err)
	}
	return out, nil
}

func (r *SeatingDraftRepo) CountItemsByGuestID(ctx context.Context, db DBTX, draftID, guestID uuid.UUID) (int, error) {
	const query = `
		SELECT COUNT(*) FROM seating_draft_items
		WHERE draft_id = $1 AND guest_id = $2
	`
	var count int
	if err := db.QueryRow(ctx, query, draftID, guestID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count draft items by guest: %w", err)
	}
	return count, nil
}

const seatingDraftPreviewSelect = `
	SELECT
		di.id,
		g.name,
		g.email,
		s.code,
		s.id,
		c.name,
		c.code,
		s.section
`

const seatingDraftPreviewBaseFrom = `
	FROM seating_draft_items di
	JOIN seating_drafts d ON d.id = di.draft_id AND d.status = 'open'
	JOIN guests g ON g.id = di.guest_id AND g.deleted_at IS NULL
	JOIN seats s ON s.id = di.seat_id AND s.deleted_at IS NULL
	JOIN seat_categories c ON c.id = di.category_id AND c.deleted_at IS NULL
	WHERE d.event_id = $1
`

func seatingDraftPreviewSearchWhere(q string, argPos int) (string, []any) {
	if strings.TrimSpace(q) == "" {
		return "", nil
	}
	pattern := SearchPattern(q)
	clause := fmt.Sprintf(
		` AND (g.name ILIKE $%d OR g.email ILIKE $%d OR s.code ILIKE $%d)`,
		argPos, argPos, argPos,
	)
	return clause, []any{pattern}
}

func scanSeatingDraftPreviewRows(rows pgx.Rows) ([]model.SeatingPreviewRow, error) {
	var out []model.SeatingPreviewRow
	for rows.Next() {
		var row model.SeatingPreviewRow
		if err := rows.Scan(
			&row.DraftItemID,
			&row.GuestName,
			&row.GuestEmail,
			&row.SeatCode,
			&row.SeatID,
			&row.CategoryName,
			&row.CategoryCode,
			&row.Section,
		); err != nil {
			return nil, fmt.Errorf("scan seating draft preview row: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("seating draft preview rows: %w", err)
	}
	return out, nil
}

func (r *SeatingDraftRepo) CountPreviewFiltered(ctx context.Context, db DBTX, eventID uuid.UUID, q string) (int, error) {
	args := []any{eventID}
	query := `SELECT COUNT(*)` + seatingDraftPreviewBaseFrom
	if searchClause, searchArgs := seatingDraftPreviewSearchWhere(q, len(args)+1); searchClause != "" {
		query += searchClause
		args = append(args, searchArgs...)
	}

	var total int
	if err := db.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count seating draft preview: %w", err)
	}
	return total, nil
}

func (r *SeatingDraftRepo) ListPreviewPaged(ctx context.Context, db DBTX, eventID uuid.UUID, pq PageQuery) ([]model.SeatingPreviewRow, error) {
	args := []any{eventID}
	query := seatingDraftPreviewSelect + seatingDraftPreviewBaseFrom
	if searchClause, searchArgs := seatingDraftPreviewSearchWhere(pq.Q, len(args)+1); searchClause != "" {
		query += searchClause
		args = append(args, searchArgs...)
	}

	offset := (pq.Page - 1) * pq.PageSize
	limitPos := len(args) + 1
	offsetPos := len(args) + 2
	query += fmt.Sprintf(` ORDER BY s.code ASC LIMIT $%d OFFSET $%d`, limitPos, offsetPos)
	args = append(args, pq.PageSize, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list seating draft preview paged: %w", err)
	}
	defer rows.Close()

	return scanSeatingDraftPreviewRows(rows)
}

func (r *SeatingDraftRepo) ListPreviewFiltered(ctx context.Context, db DBTX, eventID uuid.UUID, q string) ([]model.SeatingPreviewRow, error) {
	args := []any{eventID}
	query := seatingDraftPreviewSelect + seatingDraftPreviewBaseFrom
	if searchClause, searchArgs := seatingDraftPreviewSearchWhere(q, len(args)+1); searchClause != "" {
		query += searchClause
		args = append(args, searchArgs...)
	}
	query += ` ORDER BY s.code ASC`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list seating draft preview filtered: %w", err)
	}
	defer rows.Close()

	return scanSeatingDraftPreviewRows(rows)
}

func (r *SeatingDraftRepo) DeleteItemsByDraftID(ctx context.Context, db DBTX, draftID uuid.UUID) error {
	const query = `DELETE FROM seating_draft_items WHERE draft_id = $1`
	if _, err := db.Exec(ctx, query, draftID); err != nil {
		return fmt.Errorf("delete seating draft items: %w", err)
	}
	return nil
}
