package shared

import "time"

func LoadLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return time.UTC
	}
	return loc
}

func AnchorDateForDay(dayOfWeek int, loc *time.Location) time.Time {
	now := time.Now().In(loc)
	anchor := now.Truncate(24 * time.Hour)
	desired := time.Weekday((dayOfWeek + 1) % 7)
	for i := 0; i < 7 && anchor.Weekday() != desired; i++ {
		anchor = anchor.AddDate(0, 0, 1)
	}
	return anchor
}

func ParseTimeHHMM(t TimeHHMM) time.Time {
	parsed, err := time.Parse("15:04", string(t))
	if err != nil {
		return time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(0, 1, 1, parsed.Hour(), parsed.Minute(), 0, 0, time.UTC)
}
