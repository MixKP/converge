package scheduling

import "errors"

var ErrBookingConflict = errors.New("booking conflict: teacher already has a booking in this time range")

type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

type ConflictError struct {
	Msg string
}

func (e *ConflictError) Error() string { return e.Msg }

type NotFoundError struct {
	Msg string
}

func (e *NotFoundError) Error() string { return e.Msg }
