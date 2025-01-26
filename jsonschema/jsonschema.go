// Package jsonschema exposes the JSON Schema exporter under a dedicated import
// path while the root package retains a convenient entry point.
package jsonschema

import "github.com/rhevorn/shape"

// Document is a JSON Schema Draft 2020-12 document.
type Document = shape.JSONSchemaDocument

// UnsupportedError reports an operation that cannot be exported faithfully.
type UnsupportedError = shape.UnsupportedSchemaError

// Export converts a GoShape schema to JSON Schema Draft 2020-12.
func Export[T any](schema shape.Schema[T]) (Document, error) {
	return shape.JSONSchema(schema)
}
