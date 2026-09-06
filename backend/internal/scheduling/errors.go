package scheduling

import (
	"errors"

	"github.com/RinZ5/converge/backend/internal/shared"
)

var ErrBookingConflict = errors.New("booking conflict: teacher already has a booking in this time range")
var ErrCommuteConflict = errors.New("teacher does not have enough commute time between branches")

type ValidationError = shared.ValidationError
type ConflictError = shared.ConflictError
type NotFoundError = shared.NotFoundError
