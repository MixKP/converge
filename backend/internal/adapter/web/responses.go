package web

import (
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/gin-gonic/gin"
)

func requestID(c *gin.Context) string {
	return shared.RequestIDFromContext(c.Request.Context())
}

type MessageResponse struct {
	Message string `json:"message" example:"Availability saved successfully"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"failed to retrieve teachers"`
}
