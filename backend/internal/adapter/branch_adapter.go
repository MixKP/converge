package adapter

import (
	"context"
	"reflect"
)

// BranchCapacitySource is the inbound port the branch domain exposes for
// capacity lookups. *branch.Service satisfies it structurally; the adapter
// depends on this port rather than the concrete service, so data flows
// domain → port → adapter (mirrors TeacherRosterSource).
type BranchCapacitySource interface {
	GetCapacity(ctx context.Context, branchID int) (int, error)
}

type BranchCapacityAdapter struct {
	src BranchCapacitySource
}

func NewBranchCapacityAdapter(src BranchCapacitySource) *BranchCapacityAdapter {
	// Reject a nil source, including a typed nil (e.g. a nil *branch.Service
	// boxed in a non-nil interface). Otherwise the wrapper would produce a
	// non-nil BranchCapacityCheck whose calls panic at runtime, and CLPEngine's
	// own `branchCapacity == nil` guard would not catch it.
	if src == nil {
		panic("adapter: NewBranchCapacityAdapter requires a non-nil BranchCapacitySource")
	}
	if v := reflect.ValueOf(src); v.Kind() == reflect.Ptr && v.IsNil() {
		panic("adapter: NewBranchCapacityAdapter requires a non-nil BranchCapacitySource")
	}
	return &BranchCapacityAdapter{src: src}
}

func (a *BranchCapacityAdapter) GetCapacity(ctx context.Context, branchID int) (int, error) {
	return a.src.GetCapacity(ctx, branchID)
}
