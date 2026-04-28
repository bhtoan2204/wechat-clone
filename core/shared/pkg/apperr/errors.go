package apperr

import "net/http"

type Error struct {
	code       string
	message    string
	httpStatus int
}

func New(code, message string, httpStatus int) *Error {
	if httpStatus == 0 {
		httpStatus = http.StatusBadRequest
	}
	return &Error{code: code, message: message, httpStatus: httpStatus}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

func (e *Error) Code() string {
	if e == nil {
		return "internal_error"
	}
	return e.code
}

func (e *Error) Message() string {
	if e == nil {
		return "internal server error"
	}
	return e.message
}

func (e *Error) HTTPStatus() int {
	if e == nil || e.httpStatus == 0 {
		return http.StatusInternalServerError
	}
	return e.httpStatus
}
