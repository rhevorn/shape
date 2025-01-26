package shape

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJSONSchemaGoldenObject(t *testing.T) {
	t.Parallel()

	type user struct {
		Name string
		Role string
		Tags []string
	}
	schema := Object[user](
		Field("name", String().Min(2), func(value *user, name string) { value.Name = name }),
		Field("role", Enum("user", "admin"), func(value *user, role string) { value.Role = role }).Default("user"),
		Field("tags", Slice(String()).Max(3), func(value *user, tags []string) { value.Tags = tags }).Optional(),
	).Strict()
	document, err := JSONSchema(schema)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONGolden(t, filepath.Join("testdata", "jsonschema", "object.json"), document)
}

func TestJSONSchemaGoldenRecursive(t *testing.T) {
	t.Parallel()

	document, err := JSONSchema(treeSchema(nil))
	if err != nil {
		t.Fatal(err)
	}
	assertJSONGolden(t, filepath.Join("testdata", "jsonschema", "recursive.json"), document)
}

func assertJSONGolden(t *testing.T, path string, value any) {
	t.Helper()
	got, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("JSON differs from %s\n--- got ---\n%s--- want ---\n%s", path, got, want)
	}
}
