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
	anchor := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	desired := time.Weekday(dayOfWeek)
	for i := 0; i < 7 && anchor.Weekday() != desired; i++ {
		anchor = anchor.AddDate(0, 0, 1)
	}
	return anchor
}

func StartOfWeek(t time.Time, loc *time.Location) time.Time {
	local := t.In(loc)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return midnight.AddDate(0, 0, -int(midnight.Weekday()))
}

func ParseTimeHHMM(t TimeHHMM) time.Time {
	parsed, err := time.Parse("15:04", string(t))
	if err != nil {
		return time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(0, 1, 1, parsed.Hour(), parsed.Minute(), 0, 0, time.UTC)
}
