package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/types"
)

type Attempts uint8
type Metadata struct{ Source string }

func main() {
	ctx := context.Background()

	name := shape.String().
		IfZero("guest").
		Trim().ToLower().
		Apply(func(value string) (string, error) {
			return strings.ReplaceAll(value, " ", "-"), nil
		}).
		ApplyContext(func(ctx context.Context, value string) (string, error) {
			return value, ctx.Err()
		}).
		NotEmpty().MinLength(3).MaxLength(20).
		Pattern(`^[a-z-]+$`).Contains("shape").
		Refine(func(value string) error {
			if value == "reserved-shape" {
				return errors.New("reserved name")
			}
			return nil
		}).
		RefineContext(func(ctx context.Context, _ string) error { return ctx.Err() }).
		Label("name")
	value, err := name.TransformContext(ctx, "  Hello Shape  ")
	if err == nil {
		err = name.ValidateContext(ctx, value)
	}
	fmt.Printf("string: %q error=%v\n", value, err)

	showTransform("trim chars", shape.String().Trim("_/"), "__shape//")
	showTransform("left trim", shape.String().LTrim("_"), "__shape__")
	showTransform("right trim", shape.String().RTrim("_"), "__shape__")
	showTransform("upper", shape.String().ToUpper(), "shape")

	stringRules := []struct {
		name   string
		schema shape.Schema[string]
		value  string
	}{
		{"exact length", shape.String().Len(5), "shape"},
		{"one of", shape.String().OneOf("go", "shape"), "shape"},
		{"starts with", shape.String().StartsWith("sh"), "shape"},
		{"ends with", shape.String().EndsWith("pe"), "shape"},
		{"email", shape.String().Email(), "pong@example.com"},
		{"url", shape.String().URL(), "https://example.com"},
		{"uuid", shape.String().UUID(), "123e4567-e89b-12d3-a456-426614174000"},
		{"ip", shape.String().IP(), "127.0.0.1"},
	}
	for _, example := range stringRules {
		fmt.Printf("%s valid=%t\n", example.name, example.schema.Validate(example.value) == nil)
	}

	number := shape.Int().IfZero(5).
		Apply(func(value int) (int, error) { return value + 1, nil }).
		ApplyContext(func(ctx context.Context, value int) (int, error) { return value, ctx.Err() }).
		Min(1).Max(10).Gt(0).Gte(1).Lt(11).Lte(10).
		Between(1, 10).OneOf(5, 6).Positive().NonNegative().
		Refine(func(value int) error {
			if value%2 != 0 {
				return errors.New("must be even")
			}
			return nil
		}).
		RefineContext(func(ctx context.Context, _ int) error { return ctx.Err() }).
		Label("count")
	count, err := number.Transform(0)
	if err == nil {
		err = number.Validate(count)
	}
	fmt.Printf("number: %d error=%v\n", count, err)
	fmt.Println("negative valid:", shape.Int().Negative().Validate(-1) == nil)

	attemptsSchema := shape.Number[Attempts]().Between(1, 5)
	fmt.Println("named number valid:", attemptsSchema.Validate(3) == nil)
	fmt.Println("int64 valid:", shape.Int64().NonNegative().Validate(10) == nil)
	fmt.Println("float64 valid:", shape.Float64().Between(0, 1).Validate(0.5) == nil)

	boolean, err := shape.Bool().IfZero(true).Transform(false)
	fmt.Printf("bool: %t error=%v\n", boolean, err)

	now := time.Date(2025, time.April, 6, 12, 0, 0, 0, time.UTC)
	timestamp, err := shape.Time().IfZero(now).Transform(time.Time{})
	fmt.Printf("time: %s error=%v\n", timestamp.Format(time.RFC3339), err)

	durationSchema := shape.Duration().IfZero(types.Duration(30 * time.Second)).Positive()
	duration, err := durationSchema.Transform(0)
	if err == nil {
		err = durationSchema.Validate(duration)
	}
	fmt.Printf("duration: %s error=%v\n", duration, err)

	metadataSchema := shape.Value[Metadata]().
		IfZero(Metadata{Source: "api"}).
		Apply(func(value Metadata) (Metadata, error) { return value, nil }).
		Refine(func(value Metadata) error {
			if value.Source == "" {
				return errors.New("source is required")
			}
			return nil
		})
	metadata, err := metadataSchema.Transform(Metadata{})
	if err == nil {
		err = metadataSchema.ValidateFirst(metadata)
	}
	fmt.Printf("value: %#v error=%v\n", metadata, err)

	parsed, err := shape.ParseJSON(shape.String().Trim().NotEmpty(), []byte(`" Pong "`))
	fmt.Printf("scalar JSON: %q error=%v\n", parsed, err)
}

func showTransform(name string, schema shape.Schema[string], input string) {
	value, err := schema.Transform(input)
	fmt.Printf("%s: %q error=%v\n", name, value, err)
}
