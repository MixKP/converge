package shared

import "fmt"

type Validator[T any] func(T) error

func ValidateAll[T any](subject T, validators ...Validator[T]) error {
	for _, v := range validators {
		if err := v(subject); err != nil {
			return err
		}
	}
	return nil
}

func PositiveInt[T any](field string, get func(T) int) Validator[T] {
	return func(subject T) error {
		if get(subject) <= 0 {
			return &ValidationError{Msg: field + " must be positive"}
		}
		return nil
	}
}

func NonEmpty[T any](field string, get func(T) string) Validator[T] {
	return func(subject T) error {
		if get(subject) == "" {
			return &ValidationError{Msg: field + " must not be empty"}
		}
		return nil
	}
}

func DayOfWeekRange[T any](field string, get func(T) int) Validator[T] {
	return func(subject T) error {
		d := get(subject)
		if d < 0 || d > 6 {
			return &ValidationError{Msg: fmt.Sprintf("%s must be between 0 and 6, got %d", field, d)}
		}
		return nil
	}
}

func SliceNonEmpty[T any](field string, get func(T) int) Validator[T] {
	return func(subject T) error {
		if get(subject) == 0 {
			return &ValidationError{Msg: field + " must not be empty"}
		}
		return nil
	}
}
