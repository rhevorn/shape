package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rhevorn/shape/types"
	"github.com/rhevorn/shape/validate"
)

type Order struct{ Total int }

var identifier = validate.String().NotEmpty().MaxLength(20)

var username = validate.String().
	MinLength(3).
	Pattern(`^[a-z]+$`).
	And(identifier)

func main() {
	show("string", username.Validate("A"))
	show("number", validate.Number[uint8]().Between(1, 10).Validate(20))
	show("bool", validate.Bool().Refine(requireTrue).Validate(false))
	show("time", validate.Time().Refine(requireTime).Validate(time.Time{}))
	show("duration", validate.Duration().Positive().Validate(types.Duration(-time.Second)))

	name := ""
	show("pointer", validate.Pointer(validate.String().NotEmpty()).NotNull().Validate(&name))
	show("slice", validate.Slice(validate.Int().Positive()).NotEmpty().Unique().Validate([]int{1, -1}))
	show("map", validate.Map(validate.String().NotEmpty(), validate.Int().NonNegative()).NotNull().Validate(map[string]int{"score": -1}))

	order := validate.Value[Order]().Refine(func(value Order) error {
		if value.Total < 0 {
			return errors.New("total must not be negative")
		}
		return nil
	})
	show("custom", order.Validate(Order{Total: -1}))
	show("first", username.ValidateFirst(""))
	contextual := validate.String().RefineContext(func(ctx context.Context, value string) error {
		return ctx.Err()
	})
	show("context", contextual.ValidateContext(context.Background(), "ok"))
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
		fmt.Printf("%s: %d issue(s), first=%s\n", name, len(validationError.Issues), validationError.Issues[0].Code)
		return
	}
	fmt.Printf("%s: %v\n", name, err)
}
