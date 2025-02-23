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
	document, err := jsonschema.Export(shape.Struct[Request]())
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

func TestExportRejectsTransform(t *testing.T) {
	type Request struct {
		Name string `shape:"trim"`
	}
	_, err := jsonschema.Export(shape.Struct[Request]())
	var unsupported *jsonschema.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v", err)
	}
}

func TestExportRejectsWholeStructCallbacks(t *testing.T) {
	type Request struct{ Name string }
	schema := shape.Struct[Request]().Refine(func(Request) error { return nil })
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
	_, err := jsonschema.Export(shape.Struct[Request]())
	var unsupported *jsonschema.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v", err)
	}
}
