package teacher

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	mock.Mock
}

func (m *mockStore) AddTeacher(ctx context.Context, name, email string) (*Teacher, error) {
	args := m.Called(ctx, name, email)
	return args.Get(0).(*Teacher), args.Error(1)
}

func (m *mockStore) SetStatus(ctx context.Context, teacherID int, status string) error {
	args := m.Called(ctx, teacherID, status)
	return args.Error(0)
}

func (m *mockStore) GetActiveTeachers(ctx context.Context) ([]Teacher, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Teacher), args.Error(1)
}

func (m *mockStore) GetTeachersBySubject(ctx context.Context, subjectID int) ([]Teacher, error) {
	args := m.Called(ctx, subjectID)
	return args.Get(0).([]Teacher), args.Error(1)
}

func (m *mockStore) GetAllAvailability(ctx context.Context) ([]TeacherAvailability, error) {
	args := m.Called(ctx)
	return args.Get(0).([]TeacherAvailability), args.Error(1)
}

func (m *mockStore) GetBranches(ctx context.Context) ([]shared.Branch, error) {
	args := m.Called(ctx)
	return args.Get(0).([]shared.Branch), args.Error(1)
}

func (m *mockStore) GetSubjects(ctx context.Context) ([]shared.Subject, error) {
	args := m.Called(ctx)
	return args.Get(0).([]shared.Subject), args.Error(1)
}

func (m *mockStore) FindTeacherAvailability(ctx context.Context, teacherID int) ([]shared.WeeklySlot, error) {
	args := m.Called(ctx, teacherID)
	return args.Get(0).([]shared.WeeklySlot), args.Error(1)
}

func (m *mockStore) ReplaceWeeklyAvailability(ctx context.Context, teacherID int, slots []shared.WeeklySlot) error {
	args := m.Called(ctx, teacherID, slots)
	return args.Error(0)
}

func (m *mockStore) SaveRawSubmission(ctx context.Context, teacherID int, rawPayload []byte) error {
	args := m.Called(ctx, teacherID, rawPayload)
	return args.Error(0)
}

func tSlot(day int, start, end string) shared.WeeklySlot {
	return shared.WeeklySlot{DayOfWeek: day, Start: shared.TimeHHMM(start), End: shared.TimeHHMM(end)}
}

func TestTeacherService_SubmitAvailability_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	slots := []shared.WeeklySlot{
		tSlot(0, "09:00", "10:00"),
		tSlot(0, "11:00", "12:00"),
	}
	store.On("ReplaceWeeklyAvailability", mock.Anything, 42, slots).Return(nil)
	store.On("SaveRawSubmission", mock.Anything, 42, mock.Anything).Return(nil)

	err := svc.SubmitWeeklyAvailability(context.Background(), 42, slots)
	assert.NoError(t, err)
}

func TestTeacherService_SubmitAvailability_Overlap(t *testing.T) {
	svc := NewService(new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{
		tSlot(0, "09:00", "11:00"),
		tSlot(0, "10:30", "12:00"),
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.Contains(t, err.Error(), "overlapping")
}

func TestTeacherService_SubmitAvailability_EmptySlots(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	store.On("ReplaceWeeklyAvailability", mock.Anything, 1, []shared.WeeklySlot{}).Return(nil)
	store.On("SaveRawSubmission", mock.Anything, 1, mock.Anything).Return(nil)

	err := svc.SubmitWeeklyAvailability(context.Background(), 1, []shared.WeeklySlot{})
	assert.NoError(t, err)
}

func TestTeacherService_SubmitAvailability_ZeroTeacherID(t *testing.T) {
	svc := NewService(new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{tSlot(0, "09:00", "10:00")}
	err := svc.SubmitWeeklyAvailability(context.Background(), 0, slots)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
}

func TestTeacherService_SubmitAvailability_DayOfWeekOutOfRange(t *testing.T) {
	svc := NewService(new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{tSlot(7, "09:00", "10:00")}
	err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
}

func TestTeacherService_SubmitAvailability_EmptyStart(t *testing.T) {
	svc := NewService(new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{{DayOfWeek: 0, Start: shared.TimeHHMM(""), End: shared.TimeHHMM("10:00")}}
	err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
}

func TestTeacherService_SubmitAvailability_StartAfterEnd(t *testing.T) {
	svc := NewService(new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{tSlot(0, "10:00", "09:00")}
	err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
}

func TestTeacherService_SubmitAvailability_TouchingSlots(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	slots := []shared.WeeklySlot{
		tSlot(0, "09:00", "10:00"),
		tSlot(0, "10:00", "11:00"),
	}
	store.On("ReplaceWeeklyAvailability", mock.Anything, 1, slots).Return(nil)
	store.On("SaveRawSubmission", mock.Anything, 1, mock.Anything).Return(nil)

	err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)
	assert.NoError(t, err)
}

func TestTeacherService_SubmitAvailability_OverlapMessageFormat(t *testing.T) {
	svc := NewService(new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{
		tSlot(0, "09:00", "11:00"),
		tSlot(0, "10:30", "12:00"),
	}
	err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)

	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "overlapping slots on day 0: 09:00-11:00 and 10:30-12:00", valErr.Error())
}

func TestTeacherService_SubmitAvailability_CrossDays(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	slots := []shared.WeeklySlot{
		tSlot(0, "09:00", "11:00"),
		tSlot(1, "10:00", "12:00"),
	}
	store.On("ReplaceWeeklyAvailability", mock.Anything, 1, slots).Return(nil)
	store.On("SaveRawSubmission", mock.Anything, 1, mock.Anything).Return(nil)

	err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)
	assert.NoError(t, err)
}

func TestTeacherService_SubmitAvailability_ReplaceFails(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	slots := []shared.WeeklySlot{tSlot(0, "09:00", "10:00")}
	store.On("ReplaceWeeklyAvailability", mock.Anything, 1, slots).Return(errors.New("db error"))

	err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "replace availability")
}

func TestTeacherService_GetActiveTeachers_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	expected := []Teacher{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	store.On("GetActiveTeachers", mock.Anything).Return(expected, nil)

	teachers, err := svc.GetActiveTeachers(context.Background())
	assert.NoError(t, err)
	assert.Len(t, teachers, 2)
}

func TestTeacherService_GetActiveTeachers_Error(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	store.On("GetActiveTeachers", mock.Anything).Return(([]Teacher)(nil), errors.New("db error"))

	_, err := svc.GetActiveTeachers(context.Background())
	assert.Error(t, err)
}

func TestTeacherService_GetBranches_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	expected := []shared.Branch{{ID: 1, Name: "Main"}}
	store.On("GetBranches", mock.Anything).Return(expected, nil)

	branches, err := svc.GetBranches(context.Background())
	assert.NoError(t, err)
	assert.Len(t, branches, 1)
}

func TestTeacherService_GetSubjects_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	expected := []shared.Subject{{ID: 1, Name: "Math"}}
	store.On("GetSubjects", mock.Anything).Return(expected, nil)

	subjects, err := svc.GetSubjects(context.Background())
	assert.NoError(t, err)
	assert.Len(t, subjects, 1)
}

func TestTeacherService_GetTeachersBySubject_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	expected := []Teacher{{ID: 1, Name: "Alice"}}
	store.On("GetTeachersBySubject", mock.Anything, 1).Return(expected, nil)

	teachers, err := svc.GetTeachersBySubject(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, teachers, 1)
}

func TestTeacherService_GetAllAvailability_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	expected := []TeacherAvailability{{
		Teacher: Teacher{ID: 1, Name: "Alice"},
		Weekly:  []shared.WeeklySlot{{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")}},
	}}
	store.On("GetAllAvailability", mock.Anything).Return(expected, nil)

	avail, err := svc.GetAllAvailability(context.Background())
	assert.NoError(t, err)
	assert.Len(t, avail, 1)
}

func TestTeacherService_TeacherAvailability_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	expected := []shared.WeeklySlot{{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")}}
	store.On("FindTeacherAvailability", mock.Anything, 42).Return(expected, nil)

	slots, err := svc.TeacherAvailability(context.Background(), 42)
	assert.NoError(t, err)
	assert.Len(t, slots, 1)
	assert.Equal(t, "09:00", string(slots[0].Start))
}
