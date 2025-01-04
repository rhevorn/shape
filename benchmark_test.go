package goshape

import "testing"

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
