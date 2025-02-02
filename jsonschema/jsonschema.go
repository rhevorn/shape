// Package jsonschema exports shape schemas as JSON Schema Draft 2020-12.
package jsonschema

import (
	"encoding/json"

	"github.com/rhevorn/shape"
)

const draft202012 = "https://json-schema.org/draft/2020-12/schema"

// Document is a JSON Schema Draft 2020-12 document.
type Document map[string]any

// Bytes marshals the document as JSON.
func (d Document) Bytes() ([]byte, error) { return json.Marshal(d) }

// UnsupportedError reports an operation that cannot be exported faithfully.
type UnsupportedError = shape.UnsupportedSchemaError

// Export converts a schema to JSON Schema Draft 2020-12.
// Custom refinements and transforms return UnsupportedError instead of being
// silently lost.
func Export[T any](schema shape.Schema[T]) (Document, error) {
	document, err := shape.ExportDocument(schema)
	if err != nil {
		return nil, err
	}
	document["$schema"] = draft202012
	return Document(document), nil
}
