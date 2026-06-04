package web

type MessageResponse struct {
	Message string `json:"message" example:"Availability saved successfully"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"failed to retrieve teachers"`
}
