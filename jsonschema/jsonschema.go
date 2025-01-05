// Package jsonschema exposes the JSON Schema exporter under a dedicated import
// path while the root package retains a convenient entry point.
package jsonschema

import "github.com/rhevorn/goshape"

// Document is a JSON Schema Draft 2020-12 document.
type Document = goshape.JSONSchemaDocument

// UnsupportedError reports an operation that cannot be exported faithfully.
type UnsupportedError = goshape.UnsupportedSchemaError

// Export converts a GoShape schema to JSON Schema Draft 2020-12.
func Export[T any](schema goshape.Schema[T]) (Document, error) {
	return goshape.JSONSchema(schema)
}
