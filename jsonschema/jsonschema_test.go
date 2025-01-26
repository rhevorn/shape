package jsonschema_test

import (
	"testing"

	"github.com/rhevorn/shape"
	"github.com/rhevorn/shape/jsonschema"
)

func TestExport(t *testing.T) {
	document, err := jsonschema.Export(shape.String().Email())
	if err != nil {
		t.Fatal(err)
	}
	if document["format"] != "email" || document["$schema"] == nil {
		t.Fatalf("document = %#v", document)
	}
}
