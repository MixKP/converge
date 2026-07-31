package commute

import "github.com/RinZ5/converge/backend/internal/shared"

// UpdateCommuteRequest is the body of PATCH /commute.
type UpdateCommuteRequest struct {
	CommuteTime shared.Option[int] `json:"commute_time" binding:"required" swaggertype:"integer" example:"30"`
}

// CommuteResponse is returned when both branches are supplied.
type CommuteResponse struct {
	SourceBranch      int `json:"source_branch"      example:"1"`
	DestinationBranch int `json:"destination_branch" example:"2"`
	CommuteTime       int `json:"commute_time"       example:"30"`
}

// DefaultCommuteResponse is returned when no branches are supplied.
type DefaultCommuteResponse struct {
	CommuteTime int `json:"commute_time" example:"30"`
}
