package goshape

import (
	"context"
	"errors"
	"fmt"
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
	actual := "nil"
	if received != nil {
		actual = fmt.Sprintf("%T", received)
	}
	return Issue{
		Code:     CodeInvalidType,
		Message:  fmt.Sprintf("expected %s, received %s", expected, actual),
		Expected: expected,
		Received: actual,
	}
}

func runRefinements[T any](ctx context.Context, value T, refinements []refinement[T]) ([]Issue, error) {
	var issues []Issue
	for _, refine := range refinements {
		if err := checkContext(ctx); err != nil {
			return nil, err
		}
		if err := refine(value); err != nil {
			issues = append(issues, issuesFromError(err)...)
		}
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return issues, nil
}

func requireRefinement[T any](fn func(T) error) refinement[T] {
	if fn == nil {
		panic("goshape: refinement function must not be nil")
	}
	return fn
}
