package branch

import "github.com/RinZ5/converge/backend/internal/shared"

type Branch struct {
	ID       int    `json:"id"       example:"1"`
	Name     string `json:"name"     example:"Siam"`
	Capacity int    `json:"capacity" example:"30"`
	Status   string `json:"status"   example:"active"`
}

type UpdateCapacityRequest struct {
	Capacity shared.Option[int] `json:"capacity" binding:"required" swaggertype:"integer" example:"30"`
}

type CreateBranchRequest struct {
	Name     string `json:"name"     binding:"required" example:"Siam"`
	Capacity int    `json:"capacity" example:"30"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required" example:"deactivated"`
}
