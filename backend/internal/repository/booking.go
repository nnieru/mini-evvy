package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnieru/mini-evvy/internal/model"
)

const bookingSelectCols = `
	id, guest_id, event_id, category_id, seat_id, source, status,
	notes, paid_at, barcode, created_by, updated_by, created_at, updated_at, deleted_at
`

const bookingListSelectCols = `
	b.id, b.guest_id, b.event_id, b.category_id, b.seat_id, b.source, b.status,
	b.notes, b.paid_at, b.barcode, b.created_by, b.updated_by, b.created_at, b.updated_at, b.deleted_at
`

type BookingRepo struct {
	pool *pgxpool.Pool
}

func NewBookingRepo(pool *pgxpool.Pool) *BookingRepo {
	return &BookingRepo{pool: pool}
}

func scanBooking(row interface{ Scan(dest ...any) error }) (model.SeatBooking, error) {
	var booking model.SeatBooking
	err := row.Scan(
		&booking.ID,
		&booking.GuestID,
		&booking.EventID,
		&booking.CategoryID,
		&booking.SeatID,
		&booking.Source,
		&booking.Status,
		&booking.Notes,
		&booking.PaidAt,
		&booking.Barcode,
		&booking.CreatedBy,
		&booking.UpdatedBy,
		&booking.CreatedAt,
		&booking.UpdatedAt,
		&booking.DeletedAt,
	)
	return booking, err
}

func (r *BookingRepo) Create(ctx context.Context, db DBTX, b *model.SeatBooking) (*model.SeatBooking, error) {
	const query = `
		INSERT INTO seat_bookings (
			guest_id, event_id, category_id, seat_id, source, status,
			notes, paid_at, barcode, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING ` + bookingSelectCols

	booking, err := scanBooking(db.QueryRow(ctx, query,
		b.GuestID,
		b.EventID,
		b.CategoryID,
		b.SeatID,
		b.Source,
		b.Status,
		b.Notes,
		b.PaidAt,
		b.Barcode,
		b.CreatedBy,
		b.UpdatedBy,
	))
	if err != nil {
		return nil, fmt.Errorf("create booking: %w", err)
	}

	return &booking, nil
}

func (r *BookingRepo) GetByID(ctx context.Context, db DBTX, id uuid.UUID) (*model.SeatBooking, error) {
	const query = `SELECT ` + bookingSelectCols + `
		FROM seat_bookings
		WHERE id = $1 AND deleted_at IS NULL`

	booking, err := scanBooking(db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get booking by id: %w", err)
	}

	return &booking, nil
}

func (r *BookingRepo) GetByBarcode(ctx context.Context, db DBTX, barcode string) (*model.SeatBooking, error) {
	const query = `SELECT ` + bookingSelectCols + `
		FROM seat_bookings
		WHERE barcode = $1 AND deleted_at IS NULL`

	booking, err := scanBooking(db.QueryRow(ctx, query, barcode))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get booking by barcode: %w", err)
	}

	return &booking, nil
}

func (r *BookingRepo) GetPaidByGuestSeat(ctx context.Context, db DBTX, eventID, guestID, seatID uuid.UUID) (*model.SeatBooking, error) {
	const query = `SELECT ` + bookingSelectCols + `
		FROM seat_bookings
		WHERE event_id = $1 AND guest_id = $2 AND seat_id = $3
			AND status = 'paid' AND deleted_at IS NULL`

	booking, err := scanBooking(db.QueryRow(ctx, query, eventID, guestID, seatID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get paid booking by guest seat: %w", err)
	}

	return &booking, nil
}

func (r *BookingRepo) ListByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) ([]model.SeatBooking, error) {
	const query = `SELECT ` + bookingSelectCols + `
		FROM seat_bookings
		WHERE event_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC`

	rows, err := db.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("list bookings by event id: %w", err)
	}
	defer rows.Close()

	var out []model.SeatBooking
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		out = append(out, booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list bookings rows: %w", err)
	}

	return out, nil
}

type BookingListQuery struct {
	PaymentStatus string
	Q             string
	Page          int
	PageSize      int
}

func bookingSearchWhere(q string, argPos int) (string, []any) {
	if strings.TrimSpace(q) == "" {
		return "", nil
	}
	pattern := SearchPattern(q)
	clause := fmt.Sprintf(
		` AND (g.name ILIKE $%d OR g.email ILIKE $%d OR s.code ILIKE $%d OR COALESCE(b.barcode, '') ILIKE $%d)`,
		argPos, argPos, argPos, argPos,
	)
	return clause, []any{pattern}
}

func bookingPaymentStatusClause(paymentStatus string) string {
	switch paymentStatus {
	case "paid":
		return " AND b.status = 'paid'"
	case "unpaid":
		return " AND b.status IN ('not_paid', 'pending')"
	default:
		return " AND b.status IN ('paid', 'not_paid', 'pending')"
	}
}

const bookingListBaseFrom = `
	FROM seat_bookings b
	JOIN guests g ON g.id = b.guest_id AND g.deleted_at IS NULL
	JOIN seats s ON s.id = b.seat_id AND s.deleted_at IS NULL
	WHERE b.event_id = $1 AND b.deleted_at IS NULL
`

func scanBookingListRow(row interface{ Scan(dest ...any) error }) (model.BookingListRow, error) {
	var item model.BookingListRow
	err := row.Scan(
		&item.ID,
		&item.GuestID,
		&item.EventID,
		&item.CategoryID,
		&item.SeatID,
		&item.Source,
		&item.Status,
		&item.Notes,
		&item.PaidAt,
		&item.Barcode,
		&item.CreatedBy,
		&item.UpdatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
		&item.GuestName,
		&item.GuestEmail,
		&item.SeatCode,
	)
	return item, err
}

func (r *BookingRepo) CountByEventIDFiltered(ctx context.Context, db DBTX, eventID uuid.UUID, q BookingListQuery) (int, error) {
	args := []any{eventID}
	query := `SELECT COUNT(*)` + bookingListBaseFrom + bookingPaymentStatusClause(q.PaymentStatus)
	if searchClause, searchArgs := bookingSearchWhere(q.Q, len(args)+1); searchClause != "" {
		query += searchClause
		args = append(args, searchArgs...)
	}

	var total int
	if err := db.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count bookings by event id: %w", err)
	}
	return total, nil
}

func (r *BookingRepo) ListByEventIDPaged(ctx context.Context, db DBTX, eventID uuid.UUID, q BookingListQuery) ([]model.BookingListRow, error) {
	offset := (q.Page - 1) * q.PageSize
	args := []any{eventID}
	query := `SELECT ` + bookingListSelectCols + `, g.name, g.email, s.code` +
		bookingListBaseFrom + bookingPaymentStatusClause(q.PaymentStatus)
	if searchClause, searchArgs := bookingSearchWhere(q.Q, len(args)+1); searchClause != "" {
		query += searchClause
		args = append(args, searchArgs...)
	}
	limitPos := len(args) + 1
	offsetPos := len(args) + 2
	query += fmt.Sprintf(` ORDER BY b.created_at DESC LIMIT $%d OFFSET $%d`, limitPos, offsetPos)
	args = append(args, q.PageSize, offset)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list bookings by event id paged: %w", err)
	}
	defer rows.Close()

	var out []model.BookingListRow
	for rows.Next() {
		item, err := scanBookingListRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan booking list row: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list bookings paged rows: %w", err)
	}
	return out, nil
}

func (r *BookingRepo) CountActiveByGuestID(ctx context.Context, db DBTX, guestID uuid.UUID) (int, error) {
	const query = `
		SELECT COUNT(*) FROM seat_bookings
		WHERE guest_id = $1 AND deleted_at IS NULL AND status <> 'cancelled'
	`

	var count int
	err := db.QueryRow(ctx, query, guestID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active bookings by guest: %w", err)
	}
	return count, nil
}

func (r *BookingRepo) ListExpiredUnpaid(ctx context.Context, db DBTX, olderThan time.Time) ([]model.SeatBooking, error) {
	const query = `
		SELECT ` + bookingListSelectCols + `
		FROM seat_bookings b
		INNER JOIN seats s ON s.id = b.seat_id
		WHERE b.deleted_at IS NULL
			AND b.status IN ('pending', 'not_paid')
			AND b.created_at < $1
			AND s.status = 'reserved'
			AND s.deleted_at IS NULL
		ORDER BY b.created_at ASC
	`

	rows, err := db.Query(ctx, query, olderThan)
	if err != nil {
		return nil, fmt.Errorf("list expired unpaid bookings: %w", err)
	}
	defer rows.Close()

	var out []model.SeatBooking
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("scan expired booking: %w", err)
		}
		out = append(out, booking)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list expired unpaid rows: %w", err)
	}
	return out, nil
}

func (r *BookingRepo) Update(ctx context.Context, db DBTX, b *model.SeatBooking) (*model.SeatBooking, error) {
	const query = `
		UPDATE seat_bookings SET
			status = $1,
			notes = $2,
			paid_at = $3,
			barcode = $4,
			updated_by = $5,
			updated_at = now()
		WHERE id = $6 AND deleted_at IS NULL
		RETURNING ` + bookingSelectCols

	booking, err := scanBooking(db.QueryRow(ctx, query,
		b.Status,
		b.Notes,
		b.PaidAt,
		b.Barcode,
		b.UpdatedBy,
		b.ID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update booking: %w", err)
	}

	return &booking, nil
}

func (r *BookingRepo) SoftDelete(ctx context.Context, db DBTX, b *model.SeatBooking) (*model.SeatBooking, error) {
	const query = `
		UPDATE seat_bookings SET
			status = $1,
			updated_by = $2,
			deleted_at = now(),
			updated_at = now()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING ` + bookingSelectCols

	booking, err := scanBooking(db.QueryRow(ctx, query,
		b.Status,
		b.UpdatedBy,
		b.ID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("soft delete booking: %w", err)
	}

	return &booking, nil
}

func (r *BookingRepo) ListSeatingPreviewByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) ([]model.SeatingPreviewRow, error) {
	const query = `
		SELECT
			b.id,
			g.name,
			g.email,
			s.code,
			c.name,
			c.code,
			s.section
		FROM seat_bookings b
		JOIN guests g ON g.id = b.guest_id AND g.deleted_at IS NULL
		JOIN seats s ON s.id = b.seat_id AND s.deleted_at IS NULL
		JOIN seat_categories c ON c.id = b.category_id AND c.deleted_at IS NULL
		WHERE b.event_id = $1 AND b.deleted_at IS NULL AND b.status <> 'cancelled'
		ORDER BY s.code ASC
	`

	rows, err := db.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("list seating preview: %w", err)
	}
	defer rows.Close()

	var out []model.SeatingPreviewRow
	for rows.Next() {
		var row model.SeatingPreviewRow
		if err := rows.Scan(
			&row.BookingID,
			&row.GuestName,
			&row.GuestEmail,
			&row.SeatCode,
			&row.CategoryName,
			&row.CategoryCode,
			&row.Section,
		); err != nil {
			return nil, fmt.Errorf("scan seating preview row: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list seating preview rows: %w", err)
	}
	return out, nil
}

func (r *BookingRepo) ListActiveByEventID(ctx context.Context, db DBTX, eventID uuid.UUID) ([]model.SeatBooking, error) {
	const query = `SELECT ` + bookingSelectCols + `
		FROM seat_bookings
		WHERE event_id = $1 AND deleted_at IS NULL AND status <> 'cancelled'
		ORDER BY created_at ASC`

	rows, err := db.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("list active bookings by event id: %w", err)
	}
	defer rows.Close()

	var out []model.SeatBooking
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		out = append(out, booking)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list active bookings rows: %w", err)
	}
	return out, nil
}
