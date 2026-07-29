package branch

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/RinZ5/converge/backend/internal/shared"
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

func (m *mockStore) SetCapacity(ctx context.Context, branchID, capacity int) error {
	args := m.Called(ctx, branchID, capacity)
	return args.Error(0)
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

func TestBranchService_GetCapacity_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	store.On("GetBranchByID", mock.Anything, 1).Return(&Branch{ID: 1, Name: "Siam", Capacity: 3}, nil)

	capacity, err := svc.GetCapacity(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, 3, capacity)
}

func TestBranchService_GetCapacity_BranchNotFound_Propagates(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	store.On("GetBranchByID", mock.Anything, 99).Return(nil, errors.New("branch 99 not found"))

	_, err := svc.GetCapacity(context.Background(), 99)
	assert.Error(t, err)
}

func TestBranchService_SetCapacity_Success(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	store.On("SetCapacity", mock.Anything, 1, 30).Return(nil)

	err := svc.SetCapacity(context.Background(), 1, 30)
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func TestBranchService_SetCapacity_StoreError_Propagates(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	store.On("SetCapacity", mock.Anything, 99, 30).Return(&shared.NotFoundError{Msg: "branch 99 not found"})

	err := svc.SetCapacity(context.Background(), 99, 30)

	var notFoundErr *shared.NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestBranchService_SetCapacity_NegativeCapacity_Rejected(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	err := svc.SetCapacity(context.Background(), 1, -1)

	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	store.AssertNotCalled(t, "SetCapacity", mock.Anything, mock.Anything, mock.Anything)
}

func TestBranchService_SetCapacity_ZeroCapacity_Allowed(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	store.On("SetCapacity", mock.Anything, 1, 0).Return(nil)

	err := svc.SetCapacity(context.Background(), 1, 0)
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func TestBranchService_SetCapacity_InvalidBranchID_Rejected(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	err := svc.SetCapacity(context.Background(), 0, 30)

	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	store.AssertNotCalled(t, "SetCapacity", mock.Anything, mock.Anything, mock.Anything)
}

func TestBranchService_GetCapacity_InvalidBranchID_Rejected(t *testing.T) {
	store := new(mockStore)
	svc := NewService(store, slog.Default())

	_, err := svc.GetCapacity(context.Background(), -5)

	var valErr *shared.ValidationError
	assert.ErrorAs(t, err, &valErr)
	store.AssertNotCalled(t, "GetBranchByID", mock.Anything, mock.Anything)
}
