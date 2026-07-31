package scheduling

import "time"

const (
	MaxAlternatives       = 3
	CandidateStepDuration = 30 * time.Minute
	CandidateLookbehind   = 2 * time.Hour
	CandidateLookahead    = 2 * time.Hour

	// ProximityWeight is the per-step penalty (one CandidateStepDuration off
	// the requested time) in the ranking score. It dwarfs the 0-100 quality
	// score so the nearest valid slot always wins; quality only breaks ties
	// between equally-close slots.
	ProximityWeight = 1000
)
