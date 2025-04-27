package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/validate"
)

type Request struct {
	Name  string   `json:"name"`
	Items []string `json:"items"`
}

var requestSchema = shape.New[Request](
	shape.Field("Name", shape.String().
		ApplyContext(func(ctx context.Context, value string) (string, error) {
			return value, ctx.Err()
		}).
		RefineContext(func(ctx context.Context, _ string) error { return ctx.Err() })),
	shape.Field("Items", shape.String().Slice()),
).ApplyContext(
	func(ctx context.Context, value Request) (Request, error) {
		return value, ctx.Err()
	},
).RefineContext(
	func(ctx context.Context, _ Request) error { return ctx.Err() },
)

func main() {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := requestSchema.TransformContext(cancelled, Request{Name: "Pong"})
	fmt.Println("transform cancelled:", errors.Is(err, context.Canceled))

	err = requestSchema.ValidateContext(cancelled, Request{Name: "Pong"})
	fmt.Println("validate cancelled:", errors.Is(err, context.Canceled))

	_, err = requestSchema.ParseJSONContext(cancelled, []byte(`{"name":"Pong"}`))
	fmt.Println("parse cancelled:", errors.Is(err, context.Canceled))

	ctx := validate.WithLocale(context.Background(), validate.SimplifiedChinese)
	err = shape.String().NotEmpty().Label("名称").ValidateContext(ctx, "")
	fmt.Println("request locale:", err)

	value, err := requestSchema.TransformContext(context.Background(), Request{Name: "Pong"})
	fmt.Printf("active context: %#v error=%v\n", value, err)
}
