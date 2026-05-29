// Package apperr defines typed application errors used across the service layer.
//
// Each constructor (Validation, Conflict, NotFound) wraps a human-readable
// message with a sentinel that handlers check via errors.Is to map to the
// correct HTTP status code — without leaking the sentinel name into the
// response body.
//
// Usage:
//
//	return nil, apperr.Validation("document_number is required")  // → 400
//	return nil, apperr.Conflict("document_number already exists") // → 409
//	return nil, apperr.NotFound("account not found")              // → 404
//
//	errors.Is(err, apperr.ErrValidation) // true
package apperr

import "errors"

// Sentinel errors — use errors.Is to check kind, never compare error strings.
var (
	ErrValidation = errors.New("validation")
	ErrConflict   = errors.New("conflict")
	ErrNotFound   = errors.New("not found")
)

// appError carries a human-readable message and a sentinel for routing.
// Error() returns only the message so the sentinel name never reaches the API response.
type appError struct {
	sentinel error
	msg      string
}

func (e *appError) Error() string { return e.msg }
func (e *appError) Unwrap() error { return e.sentinel }

// Validation returns an error that signals bad input from the caller.
func Validation(msg string) error { return &appError{ErrValidation, msg} }

// Conflict returns an error that signals a uniqueness violation.
func Conflict(msg string) error { return &appError{ErrConflict, msg} }

// NotFound returns an error that signals a requested resource does not exist.
func NotFound(msg string) error { return &appError{ErrNotFound, msg} }
