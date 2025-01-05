package jsonschema_test

import (
	"testing"

	"github.com/rhevorn/goshape"
	"github.com/rhevorn/goshape/jsonschema"
)

func TestExport(t *testing.T) {
	document, err := jsonschema.Export(goshape.String().Email())
	if err != nil {
		t.Fatal(err)
	}
	if document["format"] != "email" || document["$schema"] == nil {
		t.Fatalf("document = %#v", document)
	}
}
