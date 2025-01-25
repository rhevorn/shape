package goshape

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

// SchemaMetadata documents a schema for generated specifications and tooling.
// It does not change parse behavior.
type SchemaMetadata struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Examples    []any  `json:"examples,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
	Default     any    `json:"default,omitempty"`
	HasDefault  bool   `json:"-"`
}

// MetadataProvider is implemented by schemas that expose descriptive metadata.
type MetadataProvider interface {
	Metadata() SchemaMetadata
}

type schemaMetadata struct{ value SchemaMetadata }

func (m schemaMetadata) title(value string) schemaMetadata {
	m.value.Title = value
	return m
}

func (m schemaMetadata) description(value string) schemaMetadata {
	m.value.Description = value
	return m
}

func (m schemaMetadata) example(value any) schemaMetadata {
	m.value.Examples = appendCopy(m.value.Examples, cloneMetadataValue(value))
	return m
}

func (m schemaMetadata) deprecated() schemaMetadata {
	m.value.Deprecated = true
	return m
}

func (m schemaMetadata) defaultValue(value any) schemaMetadata {
	m.value.Default = cloneMetadataValue(value)
	m.value.HasDefault = true
	return m
}

func applyMetadata(document map[string]any, metadata schemaMetadata) {
	value := metadata.value
	if value.Title != "" {
		document["title"] = value.Title
	}
	if value.Description != "" {
		document["description"] = value.Description
	}
	if len(value.Examples) != 0 {
		examples := make([]any, len(value.Examples))
		for index, example := range value.Examples {
			examples[index] = cloneMetadataValue(example)
		}
		document["examples"] = examples
	}
	if value.Deprecated {
		document["deprecated"] = true
	}
	if value.HasDefault {
		document["default"] = cloneMetadataValue(value.Default)
	}
}

func copyMetadata(metadata schemaMetadata) SchemaMetadata {
	result := metadata.value
	result.Examples = make([]any, len(metadata.value.Examples))
	for index, example := range metadata.value.Examples {
		result.Examples[index] = cloneMetadataValue(example)
	}
	if result.HasDefault {
		result.Default = cloneMetadataValue(result.Default)
	}
	return result
}

// cloneMetadataValue round-trips through JSON into the same dynamic type. In
// addition to detaching maps, slices, and pointers, this verifies at schema
// construction time that descriptive metadata can actually be exported.
func cloneMetadataValue(value any) any {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("goshape: schema metadata must be JSON-encodable: %v", err))
	}
	copyValue := reflect.New(reflect.TypeOf(value))
	if err := json.Unmarshal(encoded, copyValue.Interface()); err != nil {
		panic(fmt.Sprintf("goshape: schema metadata must be JSON-decodable: %v", err))
	}
	return copyValue.Elem().Interface()
}

// AnnotatedSchema adds metadata to any schema while preserving its parse
// behavior.
type AnnotatedSchema[T any] struct {
	schema   Schema[T]
	metadata schemaMetadata
}

// Annotate wraps schema with fluent metadata methods.
func Annotate[T any](schema Schema[T]) AnnotatedSchema[T] {
	if schema == nil {
		panic("goshape: annotated schema must not be nil")
	}
	return AnnotatedSchema[T]{schema: schema}
}

// Parse implements Schema[T].
func (s AnnotatedSchema[T]) Parse(value any) (T, error) { return s.schema.Parse(value) }

// ParseContext implements Schema[T].
func (s AnnotatedSchema[T]) ParseContext(ctx context.Context, value any) (T, error) {
	return s.schema.ParseContext(ctx, value)
}

// Title sets title metadata.
func (s AnnotatedSchema[T]) Title(value string) AnnotatedSchema[T] {
	s.metadata = s.metadata.title(value)
	return s
}

// Description sets description metadata.
func (s AnnotatedSchema[T]) Description(value string) AnnotatedSchema[T] {
	s.metadata = s.metadata.description(value)
	return s
}

// Example appends a typed example.
func (s AnnotatedSchema[T]) Example(value T) AnnotatedSchema[T] {
	s.metadata = s.metadata.example(value)
	return s
}

// Deprecated marks the schema as deprecated metadata.
func (s AnnotatedSchema[T]) Deprecated() AnnotatedSchema[T] {
	s.metadata = s.metadata.deprecated()
	return s
}

// DefaultValue sets descriptive default metadata without changing parsing.
func (s AnnotatedSchema[T]) DefaultValue(value T) AnnotatedSchema[T] {
	s.metadata = s.metadata.defaultValue(value)
	return s
}

// Metadata returns a copy of the schema metadata.
func (s AnnotatedSchema[T]) Metadata() SchemaMetadata { return copyMetadata(s.metadata) }

func (s AnnotatedSchema[T]) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	document, err := buildJSONSchemaWithContext(s.schema, ctx)
	if err != nil {
		return nil, err
	}
	applyMetadata(document, s.metadata)
	return document, nil
}
