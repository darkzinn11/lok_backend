package domain

import "errors"

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("resource conflict")
	ErrValidation   = errors.New("validation error")
)
