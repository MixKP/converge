package commute

import "context"

// ConfigStore is the driven port for the persisted, globally-configured commute
// time (minutes). Implemented by a DB adapter.
type ConfigStore interface {
	GetCommuteMinutes(ctx context.Context) (int, error)
	SetCommuteMinutes(ctx context.Context, minutes int) error
}
