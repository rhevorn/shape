package goshape

import (
	"strconv"
	"testing"
)

func FuzzStringSchema(f *testing.F) {
	for _, seed := range []string{"", "hello", "  pong  ", "用户", "a@b.example"} {
		f.Add(seed)
	}
	schema := String().Trim().Min(1).Max(128).Email()
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = schema.Parse(input)
	})
}

func FuzzIntJSON(f *testing.F) {
	for _, seed := range []string{"0", "-1", "1.0", "1e3", "9223372036854775808", "NaN"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = ParseJSON(Int64(), []byte(input))
	})
}

func FuzzGenericNumberJSON(f *testing.F) {
	for _, seed := range []string{"0", "255", "256", "-1", "1e2", "1.5", "null"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = ParseJSON(Number[uint8](), []byte(input))
		_, _ = ParseJSON(Number[float32](), []byte(input))
	})
}

func FuzzObject(f *testing.F) {
	f.Add("Pong", "pong@example.com", int64(30))
	f.Add("", "bad", int64(-1))
	schema := testUserSchema().Strict()
	f.Fuzz(func(t *testing.T, name, email string, age int64) {
		inputAge := any(age)
		if strconv.IntSize == 64 {
			inputAge = int(age)
		}
		_, _ = schema.Parse(map[string]any{
			"name":  name,
			"email": email,
			"age":   inputAge,
		})
	})
}

func FuzzParseJSON(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"name":"Pong","email":"pong@example.com","age":30}`),
		[]byte(`null`),
		[]byte(`{`),
		[]byte(`[]`),
	} {
		f.Add(seed)
	}
	schema := testUserSchema().Strict()
	f.Fuzz(func(t *testing.T, input []byte) {
		_, _ = ParseJSON(schema, input)
	})
}
