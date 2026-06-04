package teacher

import "github.com/RinZ5/converge/backend/internal/shared"

type Teacher struct {
	ID     int    `json:"id"    example:"1"`
	Name   string `json:"name"  example:"Alice"`
	Email  string `json:"email" example:"alice@example.com"`
	Status string `json:"status" example:"active"`
}

type TeacherAvailability struct {
	Teacher Teacher            `json:"teacher"`
	Weekly  []shared.WeeklySlot `json:"weekly"`
}

type AvailabilityPayload struct {
	TeacherID int                `json:"teacher_id" example:"42"`
	Weekly    []shared.WeeklySlot `json:"weekly"`
}
