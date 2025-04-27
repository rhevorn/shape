package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rhevorn/shape/types"
	"github.com/rhevorn/shape/validate"
)

type Priority uint8
type Order struct{ Total int }

func main() {
	ctx := context.Background()

	identifier := validate.String().NotEmpty().MaxLength(20)
	name := validate.String().
		MinLength(5).Len(5).
		OneOf("shape", "other").
		Pattern(`^[a-z]+$`).
		StartsWith("sh").EndsWith("pe").Contains("hap").
		Refine(func(value string) error {
			if value == "other" {
				return errors.New("reserved")
			}
			return nil
		}).
		RefineContext(func(ctx context.Context, _ string) error { return ctx.Err() }).
		And(identifier).
		Label("name")
	show("string", name.Validate("shape"))
	show("string context", name.ValidateContext(ctx, "shape"))
	show("string first", name.ValidateFirst(""))
	show("string first context", name.ValidateFirstContext(ctx, ""))
	show("string maximum", validate.String().MaxLength(3).Validate("long"))

	formats := []struct {
		name  string
		check validate.Validator[string]
		value string
	}{
		{"email", validate.String().Email(), "pong@example.com"},
		{"url", validate.String().URL(), "https://example.com"},
		{"uuid", validate.String().UUID(), "123e4567-e89b-12d3-a456-426614174000"},
		{"ip", validate.String().IP(), "127.0.0.1"},
	}
	for _, format := range formats {
		show(format.name, format.check.Validate(format.value))
	}

	number := validate.Int().
		Min(1).Max(10).Gt(0).Gte(1).Lt(11).Lte(10).
		Between(1, 10).OneOf(5, 6).Positive().NonNegative().
		Refine(func(value int) error {
			if value%2 != 0 {
				return errors.New("must be even")
			}
			return nil
		}).
		RefineContext(func(ctx context.Context, _ int) error { return ctx.Err() }).
		And(validate.Int().Max(100)).
		Label("count")
	show("number", number.Validate(6))
	show("negative", validate.Int().Negative().Validate(-1))
	show("named number", validate.Number[Priority]().Between(1, 5).Validate(3))
	show("int64", validate.Int64().NonNegative().Validate(1))
	show("float64", validate.Float64().Between(0, 1).Validate(0.5))
	show("duration", validate.Duration().Positive().Validate(types.Duration(time.Second)))

	show("bool", validate.Bool().Refine(requireTrue).Validate(true))
	show("time", validate.Time().Refine(requireTime).Validate(time.Now()))
	show("value", validate.Value[Order]().
		Refine(func(value Order) error {
			if value.Total < 0 {
				return errors.New("negative total")
			}
			return nil
		}).
		And(validate.Value[Order]()).
		Label("order").
		Validate(Order{Total: 1}))

	pointerValue := "shape"
	pointer := validate.Pointer(validate.String().NotEmpty()).
		NotNull().NotEmpty().
		Refine(func(*string) error { return nil }).
		RefineContext(func(ctx context.Context, _ *string) error { return ctx.Err() }).
		And(validate.Pointer(validate.String().MaxLength(10))).
		Label("pointer")
	show("pointer", pointer.Validate(&pointerValue))

	slice := validate.Slice(validate.Int().Positive()).
		NotNull().NotEmpty().Min(2).Max(2).Len(2).Unique().
		Refine(func([]int) error { return nil }).
		RefineContext(func(ctx context.Context, _ []int) error { return ctx.Err() }).
		And(validate.Slice(validate.Int().Max(10))).
		Label("items")
	show("slice", slice.Validate([]int{1, 2}))

	values := validate.Map(validate.String().NotEmpty(), validate.Int().NonNegative()).
		NotNull().NotEmpty().Min(1).Max(1).Len(1).
		Refine(func(map[string]int) error { return nil }).
		RefineContext(func(ctx context.Context, _ map[string]int) error { return ctx.Err() }).
		And(validate.Map(validate.String(), validate.Int().Max(10))).
		Label("scores")
	show("map", values.Validate(map[string]int{"math": 10}))
}

func requireTrue(value bool) error {
	if !value {
		return errors.New("must be true")
	}
	return nil
}

func requireTime(value time.Time) error {
	if value.IsZero() {
		return errors.New("time is required")
	}
	return nil
}

func show(name string, err error) {
	var validationError *validate.Error
	if errors.As(err, &validationError) {
		fmt.Printf("%s: %d issue(s), first=%s path=%s\n",
			name,
			len(validationError.Issues),
			validationError.Issues[0].Code,
			validationError.Issues[0].Path,
		)
		return
	}
	fmt.Printf("%s: %v\n", name, err)
}
