package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/nnieru/mini-evvy/internal/repository"
	"github.com/xuri/excelize/v2"
)

var (
	ErrInvalidGuestImportFile = errors.New("invalid guest import file")
)

var guestImportHeaders = []string{
	"Name",
	"Email",
	"Invoice Code",
	"Order Time",
	"Ticket Name",
	"Ticket Quantity",
	"Status",
}

type guestImportRow struct {
	RowNumber   int
	Name        string
	Email       string
	OrderTime   *time.Time
	CategoryCode string
	TicketCount int
	IsPaid      bool
}

type GuestImportRowError struct {
	Row     int
	Message string
}

type GuestImportResult struct {
	Created int
	Updated int
	Failed  int
	Errors  []GuestImportRowError
}

func parseGuestImportXLSX(data []byte) ([]guestImportRow, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidGuestImportFile, err)
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	if sheet == "" {
		return nil, fmt.Errorf("%w: no sheets", ErrInvalidGuestImportFile)
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("%w: read rows: %v", ErrInvalidGuestImportFile, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: empty file", ErrInvalidGuestImportFile)
	}

	header := rows[0]
	if len(header) < len(guestImportHeaders) {
		return nil, fmt.Errorf("%w: missing columns", ErrInvalidGuestImportFile)
	}
	for i, want := range guestImportHeaders {
		if strings.TrimSpace(header[i]) != want {
			return nil, fmt.Errorf("%w: expected header %q at column %d", ErrInvalidGuestImportFile, want, i+1)
		}
	}

	out := make([]guestImportRow, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		rowNum := i + 1
		cells := rows[i]
		if isEmptyRow(cells) {
			continue
		}
		for len(cells) < len(guestImportHeaders) {
			cells = append(cells, "")
		}

		name := strings.TrimSpace(cells[0])
		email := strings.TrimSpace(cells[1])
		orderTimeRaw := strings.TrimSpace(cells[3])
		categoryCode := strings.TrimSpace(cells[4])
		qtyRaw := strings.TrimSpace(cells[5])
		status := strings.TrimSpace(cells[6])

		if name == "" {
			return nil, fmt.Errorf("%w: row %d: name required", ErrInvalidGuestImportFile, rowNum)
		}
		if email == "" {
			return nil, fmt.Errorf("%w: row %d: email required", ErrInvalidGuestImportFile, rowNum)
		}
		if categoryCode == "" {
			return nil, fmt.Errorf("%w: row %d: ticket name required", ErrInvalidGuestImportFile, rowNum)
		}
		if qtyRaw == "" {
			return nil, fmt.Errorf("%w: row %d: ticket quantity required", ErrInvalidGuestImportFile, rowNum)
		}

		qty, err := strconv.Atoi(qtyRaw)
		if err != nil || qty < 1 {
			return nil, fmt.Errorf("%w: row %d: invalid ticket quantity", ErrInvalidGuestImportFile, rowNum)
		}

		var orderTime *time.Time
		isPaid := strings.EqualFold(status, "PAID")
		if isPaid && orderTimeRaw != "" {
			parsed, err := parseOrderTime(f, sheet, rowNum, orderTimeRaw)
			if err != nil {
				return nil, fmt.Errorf("%w: row %d: %v", ErrInvalidGuestImportFile, rowNum, err)
			}
			orderTime = parsed
		}

		out = append(out, guestImportRow{
			RowNumber:    rowNum,
			Name:         name,
			Email:        email,
			OrderTime:    orderTime,
			CategoryCode: categoryCode,
			TicketCount:  qty,
			IsPaid:       isPaid,
		})
	}

	return out, nil
}

func isEmptyRow(cells []string) bool {
	for _, c := range cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func parseOrderTime(f *excelize.File, sheet string, rowNum int, raw string) (*time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return &t, nil
		}
	}
	if serial, err := strconv.ParseFloat(raw, 64); err == nil {
		t, err := excelize.ExcelDateToTime(serial, false)
		if err == nil {
			return &t, nil
		}
	}
	// try cell D{row} for excel serial stored in sheet
	cell := fmt.Sprintf("D%d", rowNum)
	if v, err := f.GetCellValue(sheet, cell); err == nil && v != "" && v != raw {
		if serial, err := strconv.ParseFloat(v, 64); err == nil {
			if t, err := excelize.ExcelDateToTime(serial, false); err == nil {
				return &t, nil
			}
		}
	}
	return nil, fmt.Errorf("invalid order time %q", raw)
}

func (s *GuestService) Import(ctx context.Context, actorID, eventID uuid.UUID, fileData []byte) (*GuestImportResult, error) {
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

	if err := ensureSeatingEditable(event.SeatingPhase); err != nil {
		return nil, err
	}

	rows, err := parseGuestImportXLSX(fileData)
	if err != nil {
		return nil, err
	}

	categories, err := s.categories.ListByEventID(ctx, s.pool, eventID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	codeToID := make(map[string]uuid.UUID, len(categories))
	for _, c := range categories {
		if c.Code == nil || strings.TrimSpace(*c.Code) == "" {
			continue
		}
		codeToID[strings.ToLower(strings.TrimSpace(*c.Code))] = c.ID
	}

	result := &GuestImportResult{}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, row := range rows {
		categoryID, ok := codeToID[strings.ToLower(row.CategoryCode)]
		if !ok {
			result.Failed++
			result.Errors = append(result.Errors, GuestImportRowError{
				Row:     row.RowNumber,
				Message: fmt.Sprintf("unknown ticket name / category code: %s", row.CategoryCode),
			})
			continue
		}

		existing, err := s.guests.GetByEventNameEmailCategory(ctx, tx, eventID, categoryID, row.Name, row.Email)
		if errors.Is(err, repository.ErrNotFound) {
			guest := &model.Guest{
				EventID:     eventID,
				CategoryID:  categoryID,
				Name:        strings.TrimSpace(row.Name),
				Email:       strings.TrimSpace(row.Email),
				TicketCount: row.TicketCount,
			}
			if row.IsPaid {
				guest.PaidDate = row.OrderTime
			}
			if _, err := s.guests.Create(ctx, tx, guest); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, GuestImportRowError{
					Row:     row.RowNumber,
					Message: fmt.Sprintf("create guest: %v", err),
				})
				continue
			}
			result.Created++
			continue
		}
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, GuestImportRowError{
				Row:     row.RowNumber,
				Message: fmt.Sprintf("lookup guest: %v", err),
			})
			continue
		}

		existing.TicketCount += row.TicketCount
		if row.IsPaid && existing.PaidDate == nil {
			existing.PaidDate = row.OrderTime
		}
		if _, err := s.guests.Update(ctx, tx, existing); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, GuestImportRowError{
				Row:     row.RowNumber,
				Message: fmt.Sprintf("update guest: %v", err),
			})
			continue
		}
		result.Updated++
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}
