package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/types"
)

type Attempts uint8
type Settings struct{ Region string }

func main() {
	ctx := context.Background()

	canonical := transform.String().Trim().ToLower()
	slug := transform.String().
		IfZero("untitled").
		Trim().ToLower().
		Apply(func(value string) (string, error) {
			return strings.ReplaceAll(value, " ", "-"), nil
		}).
		ApplyContext(func(ctx context.Context, value string) (string, error) {
			return value, ctx.Err()
		}).
		Then(canonical)
	showContext("string", ctx, slug, " Hello Shape ")
	showTransform("trim chars", transform.String().Trim("_/ "), "__/ Shape /")
	showTransform("left trim", transform.String().LTrim("_"), "__shape__")
	showTransform("right trim", transform.String().RTrim("_"), "__shape__")
	showTransform("upper", transform.String().ToUpper(), "shape")

	showTransform("number", transform.Int().IfZero(5).
		Apply(func(value int) (int, error) { return value + 1, nil }).
		ApplyContext(func(ctx context.Context, value int) (int, error) { return value, ctx.Err() }).
		Then(transform.Int().Apply(func(value int) (int, error) { return value * 2, nil })), 0)
	attempts, err := transform.Number[Attempts]().IfZero(3).Transform(0)
	fmt.Printf("named number: %d error=%v\n", attempts, err)
	showTransform("int64", transform.Int64().IfZero(10), int64(0))
	showTransform("float64", transform.Float64().IfZero(0.5), float64(0))
	showTransform("bool", transform.Bool().IfZero(true), false)

	now := time.Date(2025, time.April, 6, 12, 0, 0, 0, time.UTC)
	showTransform("time", transform.Time().IfZero(now), time.Time{})
	duration, err := transform.Duration().IfZero(types.Duration(30 * time.Second)).Transform(0)
	fmt.Printf("duration: %s error=%v\n", duration, err)

	settings := transform.Value[Settings]().
		IfZero(Settings{Region: "cn"}).
		Apply(func(value Settings) (Settings, error) {
			value.Region = strings.ToUpper(value.Region)
			return value, nil
		}).
		ApplyContext(func(ctx context.Context, value Settings) (Settings, error) {
			return value, ctx.Err()
		}).
		Then(transform.Value[Settings]())
	showTransform("value", settings, Settings{})

	fallback := "guest"
	pointer := transform.Pointer(transform.String().Trim()).
		IfNull(&fallback).
		Apply(func(value *string) (*string, error) { return value, nil }).
		ApplyContext(func(ctx context.Context, value *string) (*string, error) { return value, ctx.Err() }).
		Then(transform.Pointer(transform.String().ToLower()))
	pointerValue, err := pointer.Transform(nil)
	fmt.Printf("pointer: %q error=%v\n", dereference(pointerValue), err)

	input := []string{" one ", " two "}
	slice := transform.Slice(transform.String().Trim()).
		IfNull([]string{}).
		Apply(func(value []string) ([]string, error) { return value, nil }).
		ApplyContext(func(ctx context.Context, value []string) ([]string, error) { return value, ctx.Err() }).
		Then(transform.Slice(transform.String().ToUpper()))
	showTransform("slice", slice, input)
	fmt.Printf("slice input unchanged: %q\n", input)

	values := transform.Map(transform.String().Trim().ToLower(), transform.Int()).
		IfNull(map[string]int{}).
		Apply(func(value map[string]int) (map[string]int, error) { return value, nil }).
		ApplyContext(func(ctx context.Context, value map[string]int) (map[string]int, error) { return value, ctx.Err() }).
		Then(transform.Map(transform.String(), transform.Int()))
	showTransform("map", values, map[string]int{" Score ": 1})
}

func showTransform[T any](name string, transformer transform.Transformer[T], input T) {
	value, err := transformer.Transform(input)
	fmt.Printf("%s: %#v error=%v\n", name, value, err)
}

func showContext[T any](name string, ctx context.Context, transformer transform.Transformer[T], input T) {
	value, err := transformer.TransformContext(ctx, input)
	fmt.Printf("%s: %#v error=%v\n", name, value, err)
}

func dereference(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}
