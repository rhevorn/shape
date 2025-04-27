package main

import (
	"context"
	"errors"
	"fmt"
)

import "github.com/rhevorn/shape"

func main() {
	ctx := context.Background()
	fallback := "guest"

	pointerSchema := shape.Pointer(shape.String().Trim().NotEmpty()).
		IfNull(&fallback).
		Apply(func(value *string) (*string, error) { return value, nil }).
		ApplyContext(func(ctx context.Context, value *string) (*string, error) {
			return value, ctx.Err()
		}).
		NotNull().NotEmpty().
		Refine(func(value *string) error {
			if value != nil && *value == "root" {
				return errors.New("reserved nickname")
			}
			return nil
		}).
		RefineContext(func(ctx context.Context, _ *string) error { return ctx.Err() }).
		Label("nickname")
	pointer, err := pointerSchema.TransformContext(ctx, nil)
	if err == nil {
		err = pointerSchema.ValidateContext(ctx, pointer)
	}
	fmt.Printf("pointer: %q error=%v\n", dereference(pointer), err)

	sliceSchema := shape.Slice(shape.String().Trim().ToLower().NotEmpty()).
		IfNull([]string{"general"}).
		Apply(func(values []string) ([]string, error) { return values, nil }).
		ApplyContext(func(ctx context.Context, values []string) ([]string, error) {
			return values, ctx.Err()
		}).
		NotNull().NotEmpty().Min(2).Max(2).Len(2).Unique().
		Refine(func(values []string) error {
			if len(values) > 0 && values[0] == "forbidden" {
				return errors.New("forbidden first tag")
			}
			return nil
		}).
		RefineContext(func(ctx context.Context, _ []string) error { return ctx.Err() }).
		Label("tags")
	tags, err := sliceSchema.Transform([]string{" Go ", " Shape "})
	if err == nil {
		err = sliceSchema.Validate(tags)
	}
	fmt.Printf("slice: %#v error=%v\n", tags, err)

	mapSchema := shape.Map(
		shape.String().Trim().ToLower().NotEmpty(),
		shape.Int().NonNegative(),
	).
		IfNull(map[string]int{"unknown": 0}).
		Apply(func(values map[string]int) (map[string]int, error) { return values, nil }).
		ApplyContext(func(ctx context.Context, values map[string]int) (map[string]int, error) {
			return values, ctx.Err()
		}).
		NotNull().NotEmpty().Min(1).Max(1).Len(1).
		Refine(func(map[string]int) error { return nil }).
		RefineContext(func(ctx context.Context, _ map[string]int) error { return ctx.Err() }).
		Label("scores")
	scores, err := mapSchema.Transform(map[string]int{" Docs ": 10})
	if err == nil {
		err = mapSchema.ValidateFirstContext(ctx, scores)
	}
	fmt.Printf("map: %#v error=%v\n", scores, err)

	textPointer := shape.String().Trim().Pointer()
	textSlice := shape.String().Trim().Slice()
	_, _ = textPointer.Transform(nil)
	items, _ := textSlice.Transform([]string{" one ", " two "})
	fmt.Printf("fluent composition: %#v\n", items)
}

func dereference(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}
