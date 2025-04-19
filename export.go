package shape

import (
	"errors"

	"github.com/rhevorn/shape/internal/tagged"
)

// UnsupportedSchemaError reports processing that JSON Schema cannot represent
// without changing Shape's runtime behavior.
type UnsupportedSchemaError struct{ Feature string }

// Error identifies the behavior that cannot be represented.
func (e *UnsupportedSchemaError) Error() string {
	if e == nil {
		return "shape: schema export does not support this schema"
	}
	return "shape: schema export does not support " + e.Feature
}

// ExportDocument exports representable behavior as a JSON Schema object without
// a root $schema dialect declaration. Most users should call jsonschema.Export.
func ExportDocument[T any](schema Schema[T]) (map[string]any, error) {
	provider, ok := any(schema).(interface {
		schemaPlan() (*tagged.Plan, error)
	})
	if !ok {
		return nil, &UnsupportedSchemaError{Feature: "custom schema"}
	}
	plan, err := provider.schemaPlan()
	if err != nil {
		return nil, err
	}
	document, err := tagged.Export(plan)
	if err == nil {
		return document, nil
	}
	var unsupported *tagged.UnsupportedError
	if errors.As(err, &unsupported) && unsupported != nil {
		return nil, &UnsupportedSchemaError{Feature: unsupported.Feature}
	}
	return nil, err
}
