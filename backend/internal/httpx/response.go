package httpx

import (
	"encoding/json"
	"net/http"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Envelope struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrorBody `json:"error,omitempty"`
}

func OK(w http.ResponseWriter, status int, data any) {
	write(w, status, Envelope{
		Success: true,
		Data:    data,
	})
}

func Fail(w http.ResponseWriter, status int, code, message string) {
	write(w, status, Envelope{
		Success: false,
		Error: &ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

func write(w http.ResponseWriter, status int, body Envelope) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
