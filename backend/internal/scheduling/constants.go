package scheduling

import "time"

const (
	MaxAlternatives       = 3
	CandidateStepDuration = 30 * time.Minute
	CandidateLookbehind   = 2 * time.Hour
	CandidateLookahead    = 2 * time.Hour
)
