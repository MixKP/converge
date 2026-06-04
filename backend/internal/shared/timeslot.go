package shared

import "time"

func AnchorDateForDay(dayOfWeek int, loc *time.Location) time.Time {
	now := time.Now().In(loc)
	anchor := now.Truncate(24 * time.Hour)
	desired := time.Weekday((dayOfWeek + 1) % 7)
	for anchor.Weekday() != desired {
		anchor = anchor.Add(24 * time.Hour)
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
