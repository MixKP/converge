package adapter

import (
	"context"
	"reflect"
	"time"
)

// CommuteConfigSource is the inbound port the commute domain exposes for the
// configured commute time. *commute.Service satisfies it structurally; the
// adapter depends on this port rather than the concrete service, so data flows
// domain → port → adapter (mirrors BranchCapacitySource).
type CommuteConfigSource interface {
	Minutes(ctx context.Context) (int, error)
}

// CommuteAdapter adapts the commute domain to scheduling's CommuteProvider
// port: it converts the configured minutes into the global travel-time
// duration.
type CommuteAdapter struct {
	src CommuteConfigSource
}

func NewCommuteAdapter(src CommuteConfigSource) *CommuteAdapter {
	// Reject a nil source, including a typed nil (e.g. a nil *commute.Service
	// boxed in a non-nil interface), so the wrapper can never produce a
	// non-nil CommuteProvider whose calls panic at runtime.
	if src == nil {
		panic("adapter: NewCommuteAdapter requires a non-nil CommuteConfigSource")
	}
	if v := reflect.ValueOf(src); v.Kind() == reflect.Ptr && v.IsNil() {
		panic("adapter: NewCommuteAdapter requires a non-nil CommuteConfigSource")
	}
	return &CommuteAdapter{src: src}
}

func (a *CommuteAdapter) DefaultCommute(ctx context.Context) (time.Duration, error) {
	mins, err := a.src.Minutes(ctx)
	if err != nil {
		return 0, err
	}
	return time.Duration(mins) * time.Minute, nil
}
