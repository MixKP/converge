package branch

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockStore struct {
	mock.Mock
}

func (m *mockStore) GetBranches(ctx context.Context) ([]Branch, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Branch), args.Error(1)
}

func (m *mockStore) GetBranchByID(ctx context.Context, branchID int) (*Branch, error) {
	args := m.Called(ctx, branchID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Branch), args.Error(1)
}

func (m *mockStore) CountOverlappingBookings(ctx context.Context, branchID int, startTime, endTime time.Time) (int, error) {
	args := m.Called(ctx, branchID, startTime, endTime)
	return args.Int(0), args.Error(1)
}

func TestBranchService_GetBranches_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	expected := []Branch{{ID: 1, Name: "Siam", Capacity: 30}}
	store.On("GetBranches", mock.Anything).Return(expected, nil)

	branches, err := svc.GetBranches(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expected, branches)
}

func TestBranchService_GetBranches_StoreError_Propagates(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	store.On("GetBranches", mock.Anything).Return(([]Branch)(nil), errors.New("db error"))

	_, err := svc.GetBranches(context.Background())
	assert.Error(t, err)
}

func TestBranchService_CheckCapacity_UnderCapacity(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	start := time.Now()
	end := start.Add(time.Hour)
	store.On("GetBranchByID", mock.Anything, 1).Return(&Branch{ID: 1, Name: "Siam", Capacity: 3}, nil)
	store.On("CountOverlappingBookings", mock.Anything, 1, start, end).Return(2, nil)

	ok, err := svc.CheckCapacity(context.Background(), 1, start, end)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestBranchService_CheckCapacity_AtCapacity(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	start := time.Now()
	end := start.Add(time.Hour)
	store.On("GetBranchByID", mock.Anything, 1).Return(&Branch{ID: 1, Name: "Siam", Capacity: 3}, nil)
	store.On("CountOverlappingBookings", mock.Anything, 1, start, end).Return(3, nil)

	ok, err := svc.CheckCapacity(context.Background(), 1, start, end)
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestBranchService_CheckCapacity_BranchNotFound_Propagates(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	start := time.Now()
	end := start.Add(time.Hour)
	store.On("GetBranchByID", mock.Anything, 99).Return(nil, errors.New("branch 99 not found"))

	_, err := svc.CheckCapacity(context.Background(), 99, start, end)
	assert.Error(t, err)
	store.AssertNotCalled(t, "CountOverlappingBookings", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestBranchService_CheckCapacity_CountError_Propagates(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	start := time.Now()
	end := start.Add(time.Hour)
	store.On("GetBranchByID", mock.Anything, 1).Return(&Branch{ID: 1, Name: "Siam", Capacity: 3}, nil)
	store.On("CountOverlappingBookings", mock.Anything, 1, start, end).Return(0, errors.New("db error"))

	_, err := svc.CheckCapacity(context.Background(), 1, start, end)
	assert.Error(t, err)
}
