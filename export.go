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
	var plan *tagged.Plan
	var err error
	switch builtIn := schema.(type) {
	case TaggedSpec[T]:
		plan, err = builtIn.schemaPlan()
	case *TaggedSpec[T]:
		if builtIn == nil {
			return nil, &UnsupportedSchemaError{Feature: "uninitialized schema"}
		}
		plan, err = builtIn.schemaPlan()
	default:
		return nil, &UnsupportedSchemaError{Feature: "custom schema"}
	}

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
