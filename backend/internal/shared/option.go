package shared

import "encoding/json"

type Option[T any] struct {
	value   T
	present bool
}

func Some[T any](val T) Option[T] { return Option[T]{value: val, present: true} }
func None[T any]() Option[T]      { return Option[T]{} }

func (o Option[T]) Value() (T, bool) { return o.value, o.present }
func (o Option[T]) IsSome() bool     { return o.present }
func (o Option[T]) IsNone() bool     { return !o.present }
func (o Option[T]) Or(fallback T) T {
	if o.present {
		return o.value
	}
	return fallback
}

func (o Option[T]) SQL() any {
	if !o.present {
		return nil
	}
	return o.value
}

func (o Option[T]) IfSome(fn func(T)) {
	if o.present {
		fn(o.value)
	}
}

func (o Option[T]) MarshalJSON() ([]byte, error) {
	if !o.present {
		return json.Marshal(nil)
	}
	return json.Marshal(o.value)
}

func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = Option[T]{}
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.value = v
	o.present = true
	return nil
}
