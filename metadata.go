package goshape

import "context"

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
	m.value.Examples = appendCopy(m.value.Examples, value)
	return m
}

func (m schemaMetadata) deprecated() schemaMetadata {
	m.value.Deprecated = true
	return m
}

func (m schemaMetadata) defaultValue(value any) schemaMetadata {
	m.value.Default = value
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
		document["examples"] = append([]any(nil), value.Examples...)
	}
	if value.Deprecated {
		document["deprecated"] = true
	}
	if value.HasDefault {
		document["default"] = value.Default
	}
}

func copyMetadata(metadata schemaMetadata) SchemaMetadata {
	result := metadata.value
	result.Examples = append([]any(nil), result.Examples...)
	return result
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

func (s AnnotatedSchema[T]) buildJSONSchema() (map[string]any, error) {
	document, err := buildJSONSchema(s.schema)
	if err != nil {
		return nil, err
	}
	applyMetadata(document, s.metadata)
	return document, nil
}
