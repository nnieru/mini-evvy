package repository

import "strings"

type PageQuery struct {
	Page     int
	PageSize int
	Q        string
}

func SearchPattern(q string) string {
	return "%" + strings.TrimSpace(q) + "%"
}
