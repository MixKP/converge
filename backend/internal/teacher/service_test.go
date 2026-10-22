package teacher

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	mock.Mock
}

func (m *mockStore) AddTeacher(ctx context.Context, name, email, gender string) (*Teacher, error) {
	args := m.Called(ctx, name, email, gender)
	return args.Get(0).(*Teacher), args.Error(1)
}

func (m *mockStore) SetStatus(ctx context.Context, teacherID int, status string) error {
	args := m.Called(ctx, teacherID, status)
	return args.Error(0)
}

func (m *mockStore) SetGender(ctx context.Context, teacherID int, gender string) error {
	args := m.Called(ctx, teacherID, gender)
	return args.Error(0)
}

func (m *mockStore) GetActiveTeachers(ctx context.Context) ([]Teacher, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Teacher), args.Error(1)
}

func (m *mockStore) GetAllTeachers(ctx context.Context) ([]Teacher, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Teacher), args.Error(1)
}

func (m *mockStore) GetInactiveTeachers(ctx context.Context) ([]Teacher, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Teacher), args.Error(1)
}

func (m *mockStore) GetTeachersBySubject(ctx context.Context, subjectID int) ([]Teacher, error) {
	args := m.Called(ctx, subjectID)
	return args.Get(0).([]Teacher), args.Error(1)
}

func (m *mockStore) SubjectsForTeacher(ctx context.Context, teacherID int) ([]shared.Subject, error) {
	args := m.Called(ctx, teacherID)
	return args.Get(0).([]shared.Subject), args.Error(1)
}

func (m *mockStore) ReplaceTeacherSubjects(ctx context.Context, teacherID int, subjectIDs []int) error {
	args := m.Called(ctx, teacherID, subjectIDs)
	return args.Error(0)
}

func (m *mockStore) GetAllAvailability(ctx context.Context) ([]TeacherAvailability, error) {
	args := m.Called(ctx)
	return args.Get(0).([]TeacherAvailability), args.Error(1)
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

func (m *mockStore) ReplaceWeeklyAvailabilityAndDeleteBookingsFrom(ctx context.Context, teacherID int, slots []shared.WeeklySlot, from time.Time) (int, error) {
	args := m.Called(ctx, teacherID, slots, from)
	return args.Int(0), args.Error(1)
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
	svc := NewService(store, store, store, slog.Default())

	slots := []shared.WeeklySlot{
		tSlot(0, "09:00", "10:00"),
		tSlot(0, "11:00", "12:00"),
	}
	isThisWeeksSunday := mock.MatchedBy(func(from time.Time) bool {
		local := from.In(shared.LoadLocation())
		age := time.Since(from)
		return local.Weekday() == time.Sunday &&
			local.Hour() == 0 && local.Minute() == 0 && local.Second() == 0 &&
			age >= 0 && age < 7*24*time.Hour
	})
	store.On("ReplaceWeeklyAvailabilityAndDeleteBookingsFrom", mock.Anything, 42, slots, isThisWeeksSunday).Return(3, nil)
	store.On("SaveRawSubmission", mock.Anything, 42, mock.Anything).Return(nil)

	deleted, err := svc.SubmitWeeklyAvailability(context.Background(), 42, slots)
	assert.NoError(t, err)
	assert.Equal(t, 3, deleted)
	store.AssertExpectations(t)
}

func TestTeacherService_SubmitAvailability_Overlap(t *testing.T) {
	svc := NewService(new(mockStore), new(mockStore), new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{
		tSlot(0, "09:00", "11:00"),
		tSlot(0, "10:30", "12:00"),
	}
	_, err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.Contains(t, err.Error(), "overlapping")
}

func TestTeacherService_SubmitAvailability_EmptySlots(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	store.On("ReplaceWeeklyAvailabilityAndDeleteBookingsFrom", mock.Anything, 1, []shared.WeeklySlot{}, mock.Anything).Return(0, nil)
	store.On("SaveRawSubmission", mock.Anything, 1, mock.Anything).Return(nil)

	_, err := svc.SubmitWeeklyAvailability(context.Background(), 1, []shared.WeeklySlot{})
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func TestTeacherService_SubmitAvailability_ZeroTeacherID(t *testing.T) {
	svc := NewService(new(mockStore), new(mockStore), new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{tSlot(0, "09:00", "10:00")}
	_, err := svc.SubmitWeeklyAvailability(context.Background(), 0, slots)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
}

func TestTeacherService_SubmitAvailability_DayOfWeekOutOfRange(t *testing.T) {
	svc := NewService(new(mockStore), new(mockStore), new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{tSlot(7, "09:00", "10:00")}
	_, err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
}

func TestTeacherService_SubmitAvailability_EmptyStart(t *testing.T) {
	svc := NewService(new(mockStore), new(mockStore), new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{{DayOfWeek: 0, Start: shared.TimeHHMM(""), End: shared.TimeHHMM("10:00")}}
	_, err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
}

func TestTeacherService_SubmitAvailability_StartAfterEnd(t *testing.T) {
	svc := NewService(new(mockStore), new(mockStore), new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{tSlot(0, "10:00", "09:00")}
	_, err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)

	var valErr *ValidationError
	assert.ErrorAs(t, err, &valErr)
}

func TestTeacherService_SubmitAvailability_TouchingSlots(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	slots := []shared.WeeklySlot{
		tSlot(0, "09:00", "10:00"),
		tSlot(0, "10:00", "11:00"),
	}
	store.On("ReplaceWeeklyAvailabilityAndDeleteBookingsFrom", mock.Anything, 1, slots, mock.Anything).Return(0, nil)
	store.On("SaveRawSubmission", mock.Anything, 1, mock.Anything).Return(nil)

	_, err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func TestTeacherService_SubmitAvailability_OverlapMessageFormat(t *testing.T) {
	svc := NewService(new(mockStore), new(mockStore), new(mockStore), slog.Default())

	slots := []shared.WeeklySlot{
		tSlot(0, "09:00", "11:00"),
		tSlot(0, "10:30", "12:00"),
	}
	_, err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)

	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "overlapping slots on day 0: 09:00-11:00 and 10:30-12:00", valErr.Error())
}

func TestTeacherService_SubmitAvailability_CrossDays(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	slots := []shared.WeeklySlot{
		tSlot(0, "09:00", "11:00"),
		tSlot(1, "10:00", "12:00"),
	}
	store.On("ReplaceWeeklyAvailabilityAndDeleteBookingsFrom", mock.Anything, 1, slots, mock.Anything).Return(0, nil)
	store.On("SaveRawSubmission", mock.Anything, 1, mock.Anything).Return(nil)

	_, err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func TestTeacherService_SubmitAvailability_ReplaceFails(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	slots := []shared.WeeklySlot{tSlot(0, "09:00", "10:00")}
	store.On("ReplaceWeeklyAvailabilityAndDeleteBookingsFrom", mock.Anything, 1, slots, mock.Anything).Return(0, errors.New("db error"))

	_, err := svc.SubmitWeeklyAvailability(context.Background(), 1, slots)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "replace availability")
}

func TestTeacherService_GetActiveTeachers_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	expected := []Teacher{{ID: 1, Name: "Alice", Gender: "female"}, {ID: 2, Name: "Bob", Gender: "male"}}
	store.On("GetActiveTeachers", mock.Anything).Return(expected, nil)

	teachers, err := svc.GetActiveTeachers(context.Background())
	assert.NoError(t, err)
	assert.Len(t, teachers, 2)
}

func TestTeacherService_GetActiveTeachers_Error(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	store.On("GetActiveTeachers", mock.Anything).Return(([]Teacher)(nil), errors.New("db error"))

	_, err := svc.GetActiveTeachers(context.Background())
	assert.Error(t, err)
}

func TestTeacherService_GetAllTeachers_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	expected := []Teacher{
		{ID: 1, Name: "Alice", Gender: "female", Status: "active"},
		{ID: 2, Name: "Bob", Gender: "male", Status: "deactivated"},
	}
	store.On("GetAllTeachers", mock.Anything).Return(expected, nil)

	teachers, err := svc.GetAllTeachers(context.Background())
	assert.NoError(t, err)
	assert.Len(t, teachers, 2)
	assert.Equal(t, "deactivated", teachers[1].Status)
}

func TestTeacherService_GetAllTeachers_Error(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	store.On("GetAllTeachers", mock.Anything).Return(([]Teacher)(nil), errors.New("db error"))

	_, err := svc.GetAllTeachers(context.Background())
	assert.Error(t, err)
}

func TestTeacherService_GetInactiveTeachers_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	expected := []Teacher{{ID: 3, Name: "Carol", Gender: "female", Status: "deactivated"}}
	store.On("GetInactiveTeachers", mock.Anything).Return(expected, nil)

	teachers, err := svc.GetInactiveTeachers(context.Background())
	assert.NoError(t, err)
	assert.Len(t, teachers, 1)
	assert.Equal(t, "deactivated", teachers[0].Status)
}

func TestTeacherService_GetInactiveTeachers_Error(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	store.On("GetInactiveTeachers", mock.Anything).Return(([]Teacher)(nil), errors.New("db error"))

	_, err := svc.GetInactiveTeachers(context.Background())
	assert.Error(t, err)
}

func TestTeacherService_GetSubjects_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	expected := []shared.Subject{{ID: 1, Name: "Math"}}
	store.On("GetSubjects", mock.Anything).Return(expected, nil)

	subjects, err := svc.GetSubjects(context.Background())
	assert.NoError(t, err)
	assert.Len(t, subjects, 1)
}

func TestTeacherService_GetTeachersBySubject_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	expected := []Teacher{{ID: 1, Name: "Alice", Gender: "female"}}
	store.On("GetTeachersBySubject", mock.Anything, 1).Return(expected, nil)

	teachers, err := svc.GetTeachersBySubject(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, teachers, 1)
}

func TestTeacherService_GetAllAvailability_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	expected := []TeacherAvailability{{
		Teacher: Teacher{ID: 1, Name: "Alice", Gender: "female"},
		Weekly:  []shared.WeeklySlot{{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")}},
	}}
	store.On("GetAllAvailability", mock.Anything).Return(expected, nil)

	avail, err := svc.GetAllAvailability(context.Background())
	assert.NoError(t, err)
	assert.Len(t, avail, 1)
}

func TestTeacherService_TeacherAvailability_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	expected := []shared.WeeklySlot{{DayOfWeek: 0, Start: shared.TimeHHMM("09:00"), End: shared.TimeHHMM("10:00")}}
	store.On("FindTeacherAvailability", mock.Anything, 42).Return(expected, nil)

	slots, err := svc.TeacherAvailability(context.Background(), 42)
	assert.NoError(t, err)
	assert.Len(t, slots, 1)
	assert.Equal(t, "09:00", string(slots[0].Start))
}

func TestTeacherService_AddTeacher_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	expected := &Teacher{ID: 1, Name: "Alice", Email: "alice@example.com", Gender: "female"}
	store.On("AddTeacher", mock.Anything, "Alice", "alice@example.com", "female").Return(expected, nil)

	teacherResult, err := svc.AddTeacher(context.Background(), "Alice", "alice@example.com", "female")
	assert.NoError(t, err)
	assert.Equal(t, expected, teacherResult)
	store.AssertExpectations(t)
}

func TestTeacherService_AddTeacher_StoreError_Propagates(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	store.On("AddTeacher", mock.Anything, "Alice", "alice@example.com", "robot").Return((*Teacher)(nil), &shared.ValidationError{Msg: "invalid gender"})

	_, err := svc.AddTeacher(context.Background(), "Alice", "alice@example.com", "robot")

	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.EqualError(t, err, "invalid gender")
}

func TestTeacherService_SetGender_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	store.On("SetGender", mock.Anything, 1, "male").Return(nil)

	err := svc.SetGender(context.Background(), 1, "male")
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func TestTeacherService_SetGender_StoreError_Propagates(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, store, store, slog.Default())

	store.On("SetGender", mock.Anything, 99, "male").Return(&shared.NotFoundError{Msg: "teacher 99 not found"})

	err := svc.SetGender(context.Background(), 99, "male")

	var notFoundErr *shared.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestSetTeacherSubjects(t *testing.T) {
	ctx := context.Background()

	t.Run("deduplicates before writing", func(t *testing.T) {
		store := new(mockStore)
		// The admin submits a checklist; a repeated id is the same request, and
		// the join table would reject the second row anyway.
		store.On("ReplaceTeacherSubjects", ctx, 1, []int{3, 5}).Return(nil)
		svc := NewService(store, store, store, slog.Default())

		require.NoError(t, svc.SetTeacherSubjects(ctx, 1, []int{3, 5, 3, 5}))
		store.AssertExpectations(t)
	})

	t.Run("an empty list clears the teacher's subjects", func(t *testing.T) {
		store := new(mockStore)
		store.On("ReplaceTeacherSubjects", ctx, 1, []int{}).Return(nil)
		svc := NewService(store, store, store, slog.Default())

		require.NoError(t, svc.SetTeacherSubjects(ctx, 1, nil))
		store.AssertExpectations(t)
	})

	t.Run("rejects a non-positive subject id without touching the store", func(t *testing.T) {
		store := new(mockStore)
		svc := NewService(store, store, store, slog.Default())

		err := svc.SetTeacherSubjects(ctx, 1, []int{3, 0})
		require.Error(t, err)
		var valErr *ValidationError
		require.ErrorAs(t, err, &valErr)
		store.AssertNotCalled(t, "ReplaceTeacherSubjects", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("rejects a non-positive teacher id", func(t *testing.T) {
		store := new(mockStore)
		svc := NewService(store, store, store, slog.Default())

		require.Error(t, svc.SetTeacherSubjects(ctx, 0, []int{1}))
		store.AssertNotCalled(t, "ReplaceTeacherSubjects", mock.Anything, mock.Anything, mock.Anything)
	})
}
