package service

import (
	"bytes"
	"fmt"

	"github.com/nnieru/mini-evvy/internal/model"
	"github.com/xuri/excelize/v2"
)

func formatPreviewCategory(name string, code *string) string {
	if code != nil && *code != "" {
		return fmt.Sprintf("%s (%s)", name, *code)
	}
	return name
}

func formatPreviewSection(section *string) string {
	if section == nil || *section == "" {
		return ""
	}
	return *section
}

func buildSeatingPreviewXLSX(rows []model.SeatingPreviewRow) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetName(0)
	headers := []string{"Guest", "Email", "Seat", "Category", "Section"}
	for col, header := range headers {
		cell, err := excelize.CoordinatesToCellName(col+1, 1)
		if err != nil {
			return nil, fmt.Errorf("header cell: %w", err)
		}
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return nil, fmt.Errorf("set header: %w", err)
		}
	}

	for i, row := range rows {
		values := []any{
			row.GuestName,
			row.GuestEmail,
			row.SeatCode,
			formatPreviewCategory(row.CategoryName, row.CategoryCode),
			formatPreviewSection(row.Section),
		}
		for col, value := range values {
			cell, err := excelize.CoordinatesToCellName(col+1, i+2)
			if err != nil {
				return nil, fmt.Errorf("data cell: %w", err)
			}
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return nil, fmt.Errorf("set cell: %w", err)
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write xlsx: %w", err)
	}
	return buf.Bytes(), nil
}
