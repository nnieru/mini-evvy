package httpx

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type PageParams struct {
	Page     int
	PageSize int
	Q        string
}

func ParsePageParams(r *http.Request) PageParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	return PageParams{
		Page:     page,
		PageSize: pageSize,
		Q:        q,
	}
}
