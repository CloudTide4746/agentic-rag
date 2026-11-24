package faults

import "errors"

var (
	ErrValidation      = errors.New("validation")
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("conflict")
	ErrUnauthenticated = errors.New("unauthenticated")
)
