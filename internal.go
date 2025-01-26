package shape

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

type refinement[T any] func(T) error

func appendCopy[T any](values []T, value T) []T {
	result := make([]T, len(values)+1)
	copy(result, values)
	result[len(values)] = value
	return result
}

func checkContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func contextError(err error, ctx context.Context) error {
	if current := ctx.Err(); current != nil {
		return current
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

func invalidType(expected string, received any) Issue {
	actual := typeNameOf(received)
	return Issue{
		Code:     CodeInvalidType,
		Message:  fmt.Sprintf("expected %s, received %s", expected, actual),
		Expected: expected,
		Received: actual,
	}
}

func typeNameOf(value any) string {
	if value == nil {
		return "nil"
	}
	return fmt.Sprintf("%T", value)
}

func genericTypeName[T any]() string {
	return reflect.TypeOf((*T)(nil)).Elem().String()
}

func requireImmutableDefault[T any](value T) {
	if !deeplyImmutableValue(reflect.ValueOf(value)) {
		panic("shape: reference-bearing field defaults require DefaultFunc")
	}
}

func deeplyImmutableValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Interface:
		return value.IsNil() || deeplyImmutableValue(value.Elem())
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return value.IsNil()
	case reflect.UnsafePointer:
		return value.IsNil()
	case reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if !deeplyImmutableValue(value.Index(index)) {
				return false
			}
		}
		return true
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			if !deeplyImmutableValue(value.Field(index)) {
				return false
			}
		}
		return true
	default:
		return true
	}
}

func runRefinements[T any](ctx context.Context, value T, refinements []refinement[T]) ([]Issue, error) {
	var issues []Issue
	for _, refine := range refinements {
		if err := checkContext(ctx); err != nil {
			return nil, err
		}
		refinementErr := refine(value)
		if contextErr := contextError(refinementErr, ctx); contextErr != nil {
			return nil, contextErr
		}
		if refinementErr != nil {
			var capped bool
			issues, capped = appendIssuesBounded(issues, issuesFromError(refinementErr)...)
			if capped {
				break
			}
		}
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return issues, nil
}

func requireRefinement[T any](fn func(T) error) refinement[T] {
	if fn == nil {
		panic("shape: refinement function must not be nil")
	}
	return fn
}
