package middleware

import (
	"encoding/json"
	"gateway/pkg/stackErr"
	"net/http"
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return e.Err.Error()
}

func NewAppError(code int, msg string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: msg,
		Err:     stackErr.Error(err),
	}
}

type AppHandler func(http.ResponseWriter, *http.Request) error

func (h AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err == nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if appErr, ok := err.(*AppError); ok {
		w.WriteHeader(appErr.Code)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    appErr.Code,
			"message": appErr.Message,
		})
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    http.StatusInternalServerError,
		"message": "Internal server error",
	})
}
