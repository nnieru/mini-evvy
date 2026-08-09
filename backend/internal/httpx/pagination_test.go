package httpx

import (
	"net/http/httptest"
	"testing"
)

func TestParsePageParamsDefaults(t *testing.T) {
	req := httptest.NewRequest("GET", "/items", nil)
	got := ParsePageParams(req)
	if got.Page != 1 || got.PageSize != DefaultPageSize || got.Q != "" {
		t.Fatalf("ParsePageParams() = %+v, want page=1 pageSize=%d q=\"\"", got, DefaultPageSize)
	}
}

func TestParsePageParamsClamp(t *testing.T) {
	req := httptest.NewRequest("GET", "/items?page=0&page_size=500&q=ada", nil)
	got := ParsePageParams(req)
	if got.Page != 1 || got.PageSize != MaxPageSize || got.Q != "ada" {
		t.Fatalf("ParsePageParams() = %+v", got)
	}
}
