package commute

// DefaultCommuteMinutes is the commute time returned when the source and
// destination branches differ (or when no branches are supplied).
const DefaultCommuteMinutes = 30

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
