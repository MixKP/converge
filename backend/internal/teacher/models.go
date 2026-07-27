package teacher

import "github.com/RinZ5/converge/backend/internal/shared"

type Teacher struct {
	ID     int    `json:"id"    example:"1"`
	Name   string `json:"name"  example:"Alice"`
	Email  string `json:"email" example:"alice@example.com"`
	Status string `json:"status" example:"active"`
	Gender string `json:"gender" example:"female"`
}

type TeacherAvailability struct {
	Teacher Teacher             `json:"teacher"`
	Weekly  []shared.WeeklySlot `json:"weekly"`
}

type AvailabilityPayload struct {
	TeacherID int                 `json:"teacher_id" example:"42"`
	Weekly    []shared.WeeklySlot `json:"weekly"`
}

type CreateTeacherRequest struct {
	Name   string `json:"name"   binding:"required" example:"Alice"`
	Email  string `json:"email"  binding:"required" example:"alice@example.com"`
	Gender string `json:"gender" binding:"required" example:"female"`
}
