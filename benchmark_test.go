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
