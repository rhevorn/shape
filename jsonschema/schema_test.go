package jsonschema_test

import (
	"errors"
	"testing"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/jsonschema"
)

type customJSON string

func (*customJSON) UnmarshalJSON([]byte) error { return nil }

func TestExportTaggedStruct(t *testing.T) {
	type Request struct {
		Name string `json:"name" shape:"notempty,maxlength=50"`
		Age  int    `json:"age" shape:"min=18"`
	}
	document, err := jsonschema.Export(shape.FromTags[Request]())
	if err != nil {
		t.Fatal(err)
	}
	if document["$schema"] == nil || document["type"] != "object" {
		t.Fatalf("document = %#v", document)
	}
	properties := document["properties"].(map[string]any)
	if properties["name"].(map[string]any)["maxLength"] != 50 {
		t.Fatalf("properties = %#v", properties)
	}
}

type textJSON struct{ raw string }

func (t textJSON) MarshalText() ([]byte, error) { return []byte(t.raw), nil }

// encoding/json consults the text interfaces too, so a MarshalText type is a
// custom representation and must be refused rather than described as an object.
func TestExportRejectsTextMarshalerRepresentation(t *testing.T) {
	type Doc struct {
		Value textJSON `json:"value"`
	}
	if _, err := jsonschema.Export(shape.FromTags[Doc]()); err == nil {
		t.Fatal("MarshalText type exported as its struct shape")
	}
}

// rootObject unwraps the document-level null branch, which the exporter adds
// when the whole value accepts JSON null.
func rootObject(t *testing.T, document map[string]any) map[string]any {
	t.Helper()
	if branches, ok := document["anyOf"].([]any); ok && len(branches) == 2 {
		object, ok := branches[0].(map[string]any)
		if !ok {
			t.Fatalf("first anyOf branch = %#v", branches[0])
		}
		return object
	}
	return document
}

func TestExportPointerNotnullHasNoNullBranch(t *testing.T) {
	type Doc struct {
		Name *string `json:"name" shape:"notnull"`
	}
	document, err := jsonschema.Export(shape.FromTags[Doc]())
	if err != nil {
		t.Fatal(err)
	}
	name := rootObject(t, document)["properties"].(map[string]any)["name"].(map[string]any)
	if _, ok := name["anyOf"]; ok {
		t.Fatalf("notnull pointer advertised a null branch: %#v", name)
	}
	if name["type"] != "string" {
		t.Fatalf("name = %#v", name)
	}
}

// An optional pointer adds a single null branch to the element schema.
func TestExportOptionalPointerIsNullableOnce(t *testing.T) {
	type Doc struct {
		Name *string `json:"name"`
	}
	document, err := jsonschema.Export(shape.FromTags[Doc]())
	if err != nil {
		t.Fatal(err)
	}
	name := rootObject(t, document)["properties"].(map[string]any)["name"].(map[string]any)
	branches, ok := name["anyOf"].([]any)
	if !ok || len(branches) != 2 {
		t.Fatalf("name = %#v, want a two-branch anyOf", name)
	}
	if inner, ok := branches[0].(map[string]any); !ok || inner["type"] != "string" {
		t.Fatalf("first branch = %#v", branches[0])
	}
}

func TestExportRejectsTransform(t *testing.T) {
	type Request struct {
		Name string `shape:"trim"`
	}
	_, err := jsonschema.Export(shape.FromTags[Request]())
	var unsupported *jsonschema.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v", err)
	}
}

func TestExportRejectsWholeStructCallbacks(t *testing.T) {
	type Request struct{ Name string }
	schema := shape.FromTags[Request]().Refine(func(Request) error { return nil })
	_, err := jsonschema.Export(schema)
	var unsupported *jsonschema.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v", err)
	}
}

func TestExportRejectsCustomJSONRepresentation(t *testing.T) {
	type Request struct {
		Value customJSON `json:"value"`
	}
	_, err := jsonschema.Export(shape.FromTags[Request]())
	var unsupported *jsonschema.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v", err)
	}
}
