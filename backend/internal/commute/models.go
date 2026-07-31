package commute

import "github.com/RinZ5/converge/backend/internal/shared"

// UpdateCommuteRequest is the body of PATCH /commute.
type UpdateCommuteRequest struct {
	CommuteTime shared.Option[int] `json:"commute_time" binding:"required" swaggertype:"integer" example:"30"`
}

// CommuteResponse is the commute endpoint's single response shape. The branch
// fields are omitted when absent (no-params GET and PATCH), so the JSON is
// {"commute_time":N}; with both branches supplied they are always present.
type CommuteResponse struct {
	SourceBranch      int `json:"source_branch,omitempty"      example:"1"`
	DestinationBranch int `json:"destination_branch,omitempty" example:"2"`
	CommuteTime       int `json:"commute_time"                 example:"30"`
}
