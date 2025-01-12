package goshape

import (
	"encoding/json"
	"fmt"
	"reflect"
)

const jsonSchemaDraft202012 = "https://json-schema.org/draft/2020-12/schema"

// JSONSchemaDocument is a JSON Schema Draft 2020-12 document.
type JSONSchemaDocument map[string]any

// Bytes marshals the document as JSON.
func (d JSONSchemaDocument) Bytes() ([]byte, error) { return json.Marshal(d) }

// UnsupportedSchemaError reports a schema operation that cannot be represented
// faithfully in JSON Schema.
type UnsupportedSchemaError struct{ Operation string }

func (e *UnsupportedSchemaError) Error() string {
	return "goshape: JSON Schema does not support " + e.Operation
}

type jsonSchemaNode interface {
	buildJSONSchema(*jsonSchemaBuildContext) (map[string]any, error)
}

type jsonSchemaBuildContext struct {
	definitions map[string]any
	owners      map[string]*lazyIdentity
	building    map[string]bool
}

func newJSONSchemaBuildContext() *jsonSchemaBuildContext {
	return &jsonSchemaBuildContext{
		definitions: make(map[string]any),
		owners:      make(map[string]*lazyIdentity),
		building:    make(map[string]bool),
	}
}

// JSONSchema exports schema as JSON Schema Draft 2020-12. Custom refinements
// and transforms return UnsupportedSchemaError instead of being silently lost.
func JSONSchema[T any](schema Schema[T]) (JSONSchemaDocument, error) {
	if schema == nil {
		panic("goshape: JSON Schema source must not be nil")
	}
	buildContext := newJSONSchemaBuildContext()
	document, err := buildJSONSchemaWithContext(schema, buildContext)
	if err != nil {
		return nil, err
	}
	document["$schema"] = jsonSchemaDraft202012
	if len(buildContext.definitions) != 0 {
		document["$defs"] = buildContext.definitions
	}
	return JSONSchemaDocument(document), nil
}

func buildJSONSchemaWithContext(schema any, ctx *jsonSchemaBuildContext) (map[string]any, error) {
	node, ok := schema.(jsonSchemaNode)
	if !ok {
		return nil, &UnsupportedSchemaError{Operation: fmt.Sprintf("custom schema type %T", schema)}
	}
	return node.buildJSONSchema(ctx)
}

func unsupportedIfRefined(count int) error {
	if count != 0 {
		return &UnsupportedSchemaError{Operation: "custom refinement"}
	}
	return nil
}

func applyConstraints(document map[string]any, constraints []map[string]any) {
	for _, constraint := range constraints {
		for key, value := range constraint {
			if existing, exists := document[key]; exists && !reflect.DeepEqual(existing, value) {
				allOf, _ := document["allOf"].([]any)
				allOf = append(allOf, map[string]any{key: cloneJSONValue(value)})
				document["allOf"] = allOf
				continue
			}
			document[key] = cloneJSONValue(value)
		}
	}
}

func cloneJSONValue(value any) any {
	if value == nil {
		return nil
	}
	original := reflect.ValueOf(value)
	switch original.Kind() {
	case reflect.Slice:
		copyValue := reflect.MakeSlice(original.Type(), original.Len(), original.Len())
		reflect.Copy(copyValue, original)
		return copyValue.Interface()
	case reflect.Map:
		copyValue := reflect.MakeMapWithSize(original.Type(), original.Len())
		iterator := original.MapRange()
		for iterator.Next() {
			copyValue.SetMapIndex(iterator.Key(), iterator.Value())
		}
		return copyValue.Interface()
	default:
		return value
	}
}
