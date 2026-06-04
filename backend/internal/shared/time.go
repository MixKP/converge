package shared

import (
	"encoding/json"
	"fmt"
	"time"
)

type TimeHHMM string

func (t *TimeHHMM) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.Parse("15:04", s)
	if err != nil {
		return fmt.Errorf("invalid time format %q, expected HH:MM", s)
	}
	*t = TimeHHMM(parsed.Format("15:04"))
	return nil
}

func (t TimeHHMM) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(t))
}
