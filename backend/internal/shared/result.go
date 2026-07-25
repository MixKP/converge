package shared

type Result[T any] struct {
	value T
	err   error
}

func Ok[T any](val T) Result[T]       { return Result[T]{value: val} }
func Fail[T any](err error) Result[T] { return Result[T]{err: err} }

func (r Result[T]) Unwrap() (T, error) { return r.value, r.err }
func (r Result[T]) Value() (T, bool)   { return r.value, r.err == nil }
func (r Result[T]) Error() error       { return r.err }
func (r Result[T]) IsOk() bool         { return r.err == nil }

func Map[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.err != nil {
		return Fail[U](r.err)
	}
	return Ok(fn(r.value))
}

func (r Result[T]) MapErr(fn func(error) error) Result[T] {
	if r.err != nil {
		r.err = fn(r.err)
	}
	return r
}
