package shared

import (
	"fmt"
	"unicode/utf8"
)

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

func NonNegativeInt[T any](field string, get func(T) int) Validator[T] {
	return func(subject T) error {
		if get(subject) < 0 {
			return &ValidationError{Msg: field + " must not be negative"}
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

func MaxBytes[T any](field string, max int, get func(T) string) Validator[T] {
	return func(subject T) error {
		if len(get(subject)) > max {
			return &ValidationError{Msg: fmt.Sprintf("%s must be at most %d bytes", field, max)}
		}
		return nil
	}
}

func MaxRunes[T any](field string, max int, get func(T) string) Validator[T] {
	return func(subject T) error {
		if utf8.RuneCountInString(get(subject)) > max {
			return &ValidationError{Msg: fmt.Sprintf("%s must be at most %d characters", field, max)}
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
