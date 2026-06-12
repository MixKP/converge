package shared

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadLocation_ReturnsBangkok(t *testing.T) {
	loc := LoadLocation()
	assert.Equal(t, "Asia/Bangkok", loc.String())
}

func TestAnchorDateForDay_ValidDays(t *testing.T) {
	loc := LoadLocation()

	for day := 0; day <= 6; day++ {
		anchor := AnchorDateForDay(day, loc)
		assert.Equal(t, time.Weekday(day), anchor.Weekday(),
			"expected weekday %d, got %s", day, anchor.Weekday())
		assert.Equal(t, 0, anchor.Hour())
		assert.Equal(t, 0, anchor.Minute())
		assert.Equal(t, 0, anchor.Second())
	}
}

func TestAnchorDateForDay_NegativeDay(t *testing.T) {
	loc := LoadLocation()
	anchor := AnchorDateForDay(-1, loc)
	now := time.Now().In(loc)
	// invalid weekday -1 never matches, loop slides 7 days forward to today's weekday
	assert.Equal(t, now.Weekday(), anchor.Weekday())
}

func TestAnchorDateForDay_DayOutOfRange(t *testing.T) {
	loc := LoadLocation()
	anchor := AnchorDateForDay(7, loc)
	now := time.Now().In(loc)
	// weekday 7 has no constant, loop slides 7 days forward to today's weekday
	assert.Equal(t, now.Weekday(), anchor.Weekday())
}

func TestParseTimeHHMM_Valid(t *testing.T) {
	result := ParseTimeHHMM("09:00")
	assert.Equal(t, 9, result.Hour())
	assert.Equal(t, 0, result.Minute())
	assert.Equal(t, time.UTC, result.Location())
}

func TestParseTimeHHMM_Midnight(t *testing.T) {
	result := ParseTimeHHMM("00:00")
	assert.Equal(t, 0, result.Hour())
	assert.Equal(t, 0, result.Minute())
}

func TestParseTimeHHMM_EndOfDay(t *testing.T) {
	result := ParseTimeHHMM("23:59")
	assert.Equal(t, 23, result.Hour())
	assert.Equal(t, 59, result.Minute())
}

func TestParseTimeHHMM_Empty(t *testing.T) {
	result := ParseTimeHHMM("")
	assert.Equal(t, 0, result.Hour())
	assert.Equal(t, 0, result.Minute())
}

func TestParseTimeHHMM_InvalidHour(t *testing.T) {
	result := ParseTimeHHMM("25:00")
	assert.Equal(t, 0, result.Hour())
}

func TestParseTimeHHMM_InvalidMinute(t *testing.T) {
	result := ParseTimeHHMM("12:60")
	assert.Equal(t, 0, result.Minute())
}

func TestParseTimeHHMM_InvalidFormat(t *testing.T) {
	result := ParseTimeHHMM("abc")
	assert.Equal(t, 0, result.Hour())
}

func TestTimeHHMM_UnmarshalJSON_Valid(t *testing.T) {
	var t1 TimeHHMM
	err := t1.UnmarshalJSON([]byte(`"09:00"`))
	assert.NoError(t, err)
	assert.Equal(t, TimeHHMM("09:00"), t1)
}

func TestTimeHHMM_UnmarshalJSON_InvalidFormat(t *testing.T) {
	var t1 TimeHHMM
	err := t1.UnmarshalJSON([]byte(`"25:00"`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid time format")
}

func TestTimeHHMM_UnmarshalJSON_InvalidJSON(t *testing.T) {
	var t1 TimeHHMM
	err := t1.UnmarshalJSON([]byte(`not json`))
	assert.Error(t, err)
}

func TestTimeHHMM_MarshalJSON_Valid(t *testing.T) {
	t1 := TimeHHMM("09:00")
	b, err := t1.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `"09:00"`, string(b))
}

func TestValidationError_Format(t *testing.T) {
	err := &ValidationError{Msg: "invalid input"}
	assert.Equal(t, "invalid input", err.Error())
}

func TestConflictError_Format(t *testing.T) {
	err := &ConflictError{Msg: "already exists"}
	assert.Equal(t, "already exists", err.Error())
}

func TestNotFoundError_Format(t *testing.T) {
	err := &NotFoundError{Msg: "not found"}
	assert.Equal(t, "not found", err.Error())
}

func TestContextRequestID_Roundtrip(t *testing.T) {
	ctx := context.Background()
	newCtx, id := ContextWithRequestID(ctx)

	assert.NotEmpty(t, id)
	assert.Equal(t, id, RequestIDFromContext(newCtx))
}

func TestContextRequestID_NotFound(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, RequestIDFromContext(ctx))
}
