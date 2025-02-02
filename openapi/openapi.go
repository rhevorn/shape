// Package openapi adapts shape schemas for OpenAPI 3.1 documents.
package openapi

import "github.com/rhevorn/shape"

// Schema exports an OpenAPI 3.1-compatible Schema Object. OpenAPI 3.1 aligns
// its Schema Object with JSON Schema Draft 2020-12, so the document is the
// exported schema object without a JSON Schema $schema dialect declaration.
func Schema[T any](schema shape.Schema[T]) (map[string]any, error) {
	return shape.ExportDocument(schema)
}

// JSONRequestBody builds an OpenAPI requestBody object for JSON input.
func JSONRequestBody[T any](schema shape.Schema[T], required bool) (map[string]any, error) {
	document, err := Schema(schema)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"required": required,
		"content": map[string]any{
			"application/json": map[string]any{"schema": document},
		},
	}, nil
}

// JSONResponse builds an OpenAPI response object for a JSON response body.
func JSONResponse[T any](description string, schema shape.Schema[T]) (map[string]any, error) {
	document, err := Schema(schema)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/json": map[string]any{"schema": document},
		},
	}, nil
}
