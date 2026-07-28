package scheduling

import (
	"errors"

	"github.com/RinZ5/converge/backend/internal/shared"
)

var ErrBookingConflict = errors.New("booking conflict: teacher already has a booking in this time range")
var ErrBranchCapacityExceeded = errors.New("branch has no capacity remaining for this time range")

type ValidationError = shared.ValidationError
type ConflictError = shared.ConflictError
type NotFoundError = shared.NotFoundError
