package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/types"
)

type Settings struct{ Region string }

var canonicalText = transform.String().Trim().ToLower()

var slug = transform.String().
	IfZero("untitled").
	Trim().
	ToLower().
	Apply(func(value string) (string, error) {
		return strings.ReplaceAll(value, " ", "-"), nil
	}).
	Then(canonicalText)

func main() {
	text, err := slug.Transform(" Hello Shape ")
	print("string", text, err)
	number, err := transform.Number[uint8]().IfZero(3).Transform(0)
	print("number", number, err)
	boolean, err := transform.Bool().IfZero(true).Transform(false)
	print("bool", boolean, err)
	timestamp, err := transform.Time().IfZero(time.Now()).Transform(time.Time{})
	print("time", timestamp, err)
	duration, err := transform.Duration().IfZero(types.Duration(30 * time.Second)).Transform(0)
	print("duration", duration, err)

	fallback := "guest"
	pointer, err := transform.Pointer(transform.String().Trim()).IfNull(&fallback).Transform(nil)
	print("pointer", pointer, err)
	items, err := transform.Slice(transform.String().Trim()).IfNull([]string{}).Transform([]string{" one ", " two "})
	print("slice", items, err)
	values, err := transform.Map(transform.String().Trim().ToLower(), transform.Int()).Transform(map[string]int{" Score ": 1})
	print("map", values, err)

	settings := transform.Value[Settings]().Apply(func(value Settings) (Settings, error) {
		value.Region = strings.ToUpper(value.Region)
		return value, nil
	})
	configured, err := settings.Transform(Settings{Region: "cn"})
	print("custom", configured, err)
	contextual := transform.String().ApplyContext(func(ctx context.Context, value string) (string, error) {
		return value, ctx.Err()
	})
	text, err = contextual.TransformContext(context.Background(), "context")
	print("context", text, err)
}

func print[T any](name string, value T, err error) {
	fmt.Printf("%s: %#v error=%v\n", name, value, err)
}
