package shared

type WeeklySlot struct {
	DayOfWeek int      `json:"day_of_week" example:"0"` // 0=Monday ... 6=Sunday
	Start     TimeHHMM `json:"start"       example:"09:00"`
	End       TimeHHMM `json:"end"         example:"10:00"`
}
