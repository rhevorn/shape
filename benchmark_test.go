package shape_test

import (
	"testing"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/validate"
)

func BenchmarkSchemaParseJSON(b *testing.B) {
	type Request struct {
		Name string `json:"name" shape:"trim,notempty,maxlength=50"`
		Age  int    `json:"age" shape:"min=18"`
	}
	schema := shape.Struct[Request]()
	data := []byte(`{"name":" Pong ","age":20}`)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := schema.ParseJSON(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBasicPackages(b *testing.B) {
	transformer := transform.String().Trim().ToLower()
	validator := validate.String().NotEmpty().MaxLength(50)
	b.ReportAllocs()
	for b.Loop() {
		value, err := transformer.Transform(" Pong ")
		if err != nil {
			b.Fatal(err)
		}
		if err := validator.Validate(value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLargeSchemaTransform(b *testing.B) {
	type Large struct {
		Payload []byte
	}
	input := Large{Payload: make([]byte, 64<<10)}
	step := func(value Large) (Large, error) {
		value.Payload[0]++
		return value, nil
	}

	one := shape.New[Large]().Apply(step)
	many := shape.New[Large]().
		Apply(step).
		Apply(step).
		Apply(step).
		Apply(step).
		Apply(step).
		Apply(step).
		Apply(step).
		Apply(step)

	b.Run("one-apply", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := one.Transform(input); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("eight-chained-applies", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := many.Transform(input); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkLargeTransformerThen(b *testing.B) {
	type Large struct {
		Payload []byte
	}
	input := Large{Payload: make([]byte, 64<<10)}
	step := transform.Value[Large]().Apply(func(value Large) (Large, error) {
		value.Payload[0]++
		return value, nil
	})
	many := step.Then(step, step, step, step, step, step, step)

	b.Run("one-transformer", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := step.Transform(input); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("eight-composed-transformers", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := many.Transform(input); err != nil {
				b.Fatal(err)
			}
		}
	})
}
