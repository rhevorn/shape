package shape

import (
	"bytes"
	"encoding/json"
	"reflect"
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
	for _, seed := range []string{"0", "-1", "1.0", "1e3", "1000e-2", "1e600000000", "1e-600000000", "9223372036854775808", "NaN"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = Parse(Int64(), []byte(input))
	})
}

func FuzzGenericNumberJSON(f *testing.F) {
	for _, seed := range []string{"0", "255", "256", "-1", "1e2", "1e600000000", "1e-600000000", "1.5", "null"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = Parse(Number[uint8](), []byte(input))
		_, _ = Parse(Number[float32](), []byte(input))
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

func FuzzParse(f *testing.F) {
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
		_, _ = Parse(schema, input)
	})
}

func FuzzTupleJSON(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`[1,2,"point"]`),
		[]byte(`[]`),
		[]byte(`null`),
	} {
		f.Add(seed)
	}
	schema := coordinateSchema()
	f.Fuzz(func(t *testing.T, input []byte) {
		_, _ = Parse(schema, input)
	})
}

func FuzzMapJSON(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"ONE":1,"two":2}`),
		[]byte(`{}`),
		[]byte(`[]`),
	} {
		f.Add(seed)
	}
	schema := Map(String().ToLower().NonEmpty(), Int())
	f.Fuzz(func(t *testing.T, input []byte) {
		_, _ = Parse(schema, input)
	})
}

func FuzzLazyJSON(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"value":"root","children":[{"value":"leaf"}]}`),
		[]byte(`{"value":"root"}`),
		[]byte(`null`),
	} {
		f.Add(seed)
	}
	schema := treeSchema(nil)
	f.Fuzz(func(t *testing.T, input []byte) {
		_, _ = Parse(schema, input)
	})
}

func FuzzJSONSchemaExport(f *testing.F) {
	for _, seed := range [][3]byte{{0, 0, 0}, {1, 1, 10}, {4, 10, 1}, {7, 255, 255}} {
		f.Add(seed[0], seed[1], seed[2])
	}
	f.Fuzz(func(t *testing.T, kind, first, second byte) {
		minimum := int(first % 32)
		maximum := minimum + int(second%32)
		switch kind % 8 {
		case 0:
			assertJSONSchemaInvariant(t, String().Min(minimum).Max(maximum))
		case 1:
			assertJSONSchemaInvariant(t, Slice(Number[uint16]().Max(uint16(maximum))).Max(maximum))
		case 2:
			assertJSONSchemaInvariant(t, Map(String().Min(minimum), Bool()).Max(maximum))
		case 3:
			assertJSONSchemaInvariant(t, coordinateSchema())
		case 4:
			assertJSONSchemaInvariant(t, treeSchema(nil))
		case 5:
			assertJSONSchemaInvariant(t, OneOf[string](String().Email(), UUID()))
		case 6:
			assertJSONSchemaInvariant(t, testUserSchema().Strict())
		case 7:
			assertJSONSchemaInvariant(t, Map(String(), String()).Min(minimum).Max(maximum))
		}
	})
}

func FuzzParseReaderLimit(f *testing.F) {
	for _, seed := range []struct {
		data  []byte
		limit uint16
	}{
		{[]byte(`{"name":"Pong","email":"pong@example.com","age":30}`), 1024},
		{[]byte(`null`), 4},
		{[]byte(`{`), 1},
	} {
		f.Add(seed.data, seed.limit)
	}
	schema := testUserSchema().Strict()
	f.Fuzz(func(t *testing.T, input []byte, limit uint16) {
		_, _ = ParseReaderLimit(schema, bytes.NewReader(input), int64(limit))
	})
}

func assertJSONSchemaInvariant[T any](t *testing.T, schema Schema[T]) {
	t.Helper()
	first, err := ExportDocument(schema)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(first)
	if err != nil || !json.Valid(encoded) {
		t.Fatalf("invalid JSON Schema: %s, %v", encoded, err)
	}
	second, err := ExportDocument(schema)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("export is nondeterministic: %#v != %#v", first, second)
	}
	first["x-mutated"] = true
	if _, exists := second["x-mutated"]; exists {
		t.Fatal("exported documents share root mutation state")
	}
}
