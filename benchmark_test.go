package shape

import (
	"encoding/json"
	"testing"
)

func BenchmarkStringValidation(b *testing.B) {
	schema := String().Trim().Min(3).Max(50)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := schema.Parse("  hello  "); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEmailValidation(b *testing.B) {
	schema := String().Email()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := schema.Parse("pong@example.com"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIntegerBounds(b *testing.B) {
	schema := Int().Min(18).Max(120)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := schema.Parse(30); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenericNamedNumber(b *testing.B) {
	type score int32
	schema := Number[score]().Min(0).Max(100)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := schema.Parse(score(75)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSliceValidation(b *testing.B) {
	schema := Slice(String().Min(1)).Min(1).Max(10)
	input := []any{"one", "two", "three", "four"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := schema.Parse(input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNestedObjectValidation(b *testing.B) {
	schema := testUserSchema().Strict()
	input := map[string]any{
		"name":  "Pong",
		"email": "pong@example.com",
		"age":   30,
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := schema.Parse(input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInvalidObjectValidation(b *testing.B) {
	schema := testUserSchema().Strict()
	input := map[string]any{
		"name":    "x",
		"email":   "invalid",
		"age":     10,
		"unknown": true,
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := schema.Parse(input); err == nil {
			b.Fatal("expected validation error")
		}
	}
}

func BenchmarkRecursiveValidation(b *testing.B) {
	schema := treeSchema(nil)
	input := map[string]any{
		"value": "root",
		"children": []any{
			map[string]any{"value": "branch", "children": []any{
				map[string]any{"value": "leaf"},
			}},
		},
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := schema.Parse(input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUniqueComparableValidation(b *testing.B) {
	input := make([]any, 1000)
	for index := range input {
		input[index] = index
	}
	schema := Slice(Int()).Unique()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := schema.Parse(input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRejectHugeIntegerExponent(b *testing.B) {
	schema := Int64()
	input := json.Number("1e600000000")
	b.ReportAllocs()
	for b.Loop() {
		if _, err := schema.Parse(input); err == nil {
			b.Fatal("expected validation error")
		}
	}
}
