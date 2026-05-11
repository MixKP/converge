package models

type Teacher struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type WeeklySlot struct {
	DayOfWeek int    `json:"day_of_week"`
	Start     string `json:"start"`
	End       string `json:"end"`
}

type AvailabilityPayload struct {
	TeacherID int          `json:"teacher_id"`
	Weekly    []WeeklySlot `json:"weekly"`
}
