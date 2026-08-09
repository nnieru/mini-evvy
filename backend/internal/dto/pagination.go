package dto

type PaginatedList[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

func NewPaginatedList[T any](items []T, total, page, pageSize int) PaginatedList[T] {
	if items == nil {
		items = []T{}
	}
	return PaginatedList[T]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}
