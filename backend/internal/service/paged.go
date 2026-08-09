package service

type PagedResult[T any] struct {
	Items    []T
	Total    int
	Page     int
	PageSize int
}
