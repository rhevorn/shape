package shape

import (
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/rhevorn/shape/validate"
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
	provider, ok := any(schema).(interface{ schemaPlan() (*tagPlan, error) })
	if !ok {
		return nil, &UnsupportedSchemaError{Feature: "custom schema"}
	}
	p, err := provider.schemaPlan()
	if err != nil {
		return nil, err
	}
	return exportTagPlan(p)
}
func exportTagPlan(p *tagPlan) (map[string]any, error) { return exportTagPlanAt(p, false) }

// exportTagPlanAt builds the document for p. pointee is true when p sits behind a
// pointer: there a JSON null decodes to the pointer, not to p, so p must not
// offer a null branch of its own.
func exportTagPlanAt(p *tagPlan, pointee bool) (map[string]any, error) {
	if p == nil {
		return nil, &UnsupportedSchemaError{Feature: "uninitialized schema"}
	}
	if p.fallbackKind != "" {
		return nil, &UnsupportedSchemaError{Feature: p.fallbackKind + " fallback"}
	}
	if p.hasTransform {
		return nil, &UnsupportedSchemaError{Feature: "transform"}
	}
	t := p.typ
	var document map[string]any
	if t == durationType {
		document = map[string]any{"type": "string", "format": "duration"}
		if len(p.descriptors) != 0 {
			return nil, &UnsupportedSchemaError{Feature: "duration comparison rules"}
		}
		return applyExportRules(document, p, pointee)
	}
	if t == reflect.TypeFor[time.Time]() {
		document = map[string]any{"type": "string", "format": "date-time"}
		return applyExportRules(document, p, pointee)
	}
	if hasCustomJSON(t) {
		return nil, &UnsupportedSchemaError{Feature: fmt.Sprintf("custom JSON representation for %v", t)}
	}
	switch t.Kind() {
	case reflect.String:
		document = map[string]any{"type": "string"}
	case reflect.Bool:
		document = map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		document = map[string]any{"type": "integer"}
		if t.Kind() >= reflect.Uint && t.Kind() <= reflect.Uint64 {
			document["minimum"] = uint64(0)
			document["maximum"] = unsignedMaximum(t.Bits())
		} else {
			minimum, maximum := signedRange(t.Bits())
			document["minimum"] = minimum
			document["maximum"] = maximum
		}
	case reflect.Float32, reflect.Float64:
		document = map[string]any{"type": "number"}
	case reflect.Pointer:
		if p.element == nil {
			return nil, &UnsupportedSchemaError{Feature: "custom pointer element"}
		}
		inner, err := exportTagPlanAt(p.element, true)
		if err != nil {
			return nil, err
		}
		document = inner
	case reflect.Slice:
		if p.element == nil {
			return nil, &UnsupportedSchemaError{Feature: "custom slice element"}
		}
		inner, err := exportTagPlan(p.element)
		if err != nil {
			return nil, err
		}
		document = map[string]any{"type": "array", "items": inner}
	case reflect.Map:
		if p.element == nil {
			return nil, &UnsupportedSchemaError{Feature: "custom map element"}
		}
		if t.Key().Kind() != reflect.String {
			return nil, &UnsupportedSchemaError{Feature: "non-string map key"}
		}
		inner, err := exportTagPlan(p.element)
		if err != nil {
			return nil, err
		}
		document = map[string]any{"type": "object", "additionalProperties": inner}
	case reflect.Struct:
		if p.fields == nil {
			return nil, &UnsupportedSchemaError{Feature: "explicit Object fields"}
		}
		properties := make(map[string]any, len(p.fields))
		required := make([]string, 0, len(p.fields))
		for _, field := range p.fields {
			child, err := exportTagPlan(field.plan)
			if err != nil {
				return nil, err
			}
			properties[field.name] = child
			if !planAcceptsZero(field.plan) {
				required = append(required, field.name)
			}
		}
		document = map[string]any{"type": "object", "properties": properties}
		if len(required) != 0 {
			document["required"] = required
		}
	default:
		return nil, &UnsupportedSchemaError{Feature: fmt.Sprintf("schema %v", t)}
	}
	return applyExportRules(document, p, pointee)
}

var (
	jsonMarshalerType   = reflect.TypeFor[json.Marshaler]()
	jsonUnmarshalerType = reflect.TypeFor[json.Unmarshaler]()
	textMarshalerType   = reflect.TypeFor[encoding.TextMarshaler]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
)

// hasCustomJSON reports whether encoding/json uses a user-supplied
// representation for t. The text interfaces count: encoding/json consults
// MarshalText/UnmarshalText as well as the JSON ones, so a type with only
// MarshalText encodes as a string while the struct walk below would describe
// its fields. Missing that was a silent omission rather than a refusal.
func hasCustomJSON(t reflect.Type) bool {
	if implementsJSONCodec(t) {
		return true
	}
	return t.Kind() != reflect.Pointer && implementsJSONCodec(reflect.PointerTo(t))
}

func implementsJSONCodec(t reflect.Type) bool {
	return t.Implements(jsonMarshalerType) ||
		t.Implements(jsonUnmarshalerType) ||
		t.Implements(textMarshalerType) ||
		t.Implements(textUnmarshalerType)
}

func applyExportRules(document map[string]any, p *tagPlan, pointee bool) (map[string]any, error) {
	for _, descriptor := range p.descriptors {
		var key string
		var value any
		name, args := descriptor.Name, descriptor.Args
		switch name {
		case "min":
			if p.typ.Kind() == reflect.String {
				// min is a numeric or collection rule; the compiler rejects it on
				// a string, so reaching here means the rule set widened. Emitting
				// a document under an empty key would be silent corruption.
				return nil, &UnsupportedSchemaError{Feature: "rule min on " + p.typ.String()}
			} else if p.typ.Kind() == reflect.Slice {
				key = "minItems"
			} else if p.typ.Kind() == reflect.Map {
				key = "minProperties"
			} else {
				key = "minimum"
			}
			value = args[0]
		case "max":
			if p.typ.Kind() == reflect.Slice {
				key = "maxItems"
			} else if p.typ.Kind() == reflect.Map {
				key = "maxProperties"
			} else {
				key = "maximum"
			}
			value = args[0]
		case "minlength":
			key, value = "minLength", args[0]
		case "maxlength":
			key, value = "maxLength", args[0]
		case "len":
			switch p.typ.Kind() {
			case reflect.Slice:
				document["minItems"], document["maxItems"] = args[0], args[0]
			case reflect.Map:
				document["minProperties"], document["maxProperties"] = args[0], args[0]
			default:
				document["minLength"], document["maxLength"] = args[0], args[0]
			}
			continue
		case "gt":
			key, value = "exclusiveMinimum", args[0]
		case "gte":
			key, value = "minimum", args[0]
		case "lt":
			key, value = "exclusiveMaximum", args[0]
		case "lte":
			key, value = "maximum", args[0]
		case "positive":
			key, value = "exclusiveMinimum", 0
		case "negative":
			key, value = "exclusiveMaximum", 0
		case "nonnegative":
			key, value = "minimum", 0
		case "oneof":
			key, value = "enum", args
		case "between":
			document["minimum"], document["maximum"] = args[0], args[1]
			continue
		case "pattern":
			key, value = "pattern", args[0]
		case "email", "url", "uuid", "ip":
			format := name
			if format == "url" {
				format = "uri"
			}
			key, value = "format", format
		case "notempty":
			switch p.typ.Kind() {
			case reflect.String:
				key, value = "minLength", 1
			case reflect.Slice:
				key, value = "minItems", 1
			case reflect.Map:
				key, value = "minProperties", 1
			case reflect.Pointer:
				continue
			default:
				continue
			}
		case "notnull":
			// The non-null branch already excludes null. Whether a null branch is
			// added is decided from the complete plan below.
			continue
		case "unique":
			key, value = "uniqueItems", true
		default:
			return nil, &UnsupportedSchemaError{Feature: "rule " + name}
		}
		document[key] = value
	}
	if !pointee && planAcceptsZero(p) {
		return map[string]any{
			"anyOf": []any{document, map[string]any{"type": "null"}},
		}, nil
	}
	return document, nil
}

func planAcceptsZero(p *tagPlan) bool {
	if p == nil {
		return false
	}
	issues := make([]validate.Issue, 0)
	_, err := validatePlan(context.Background(), p, reflect.Zero(p.typ), nil, 0, false, &issues)
	return err == nil && len(issues) == 0
}

func signedRange(bits int) (int64, int64) {
	if bits == 64 {
		return -1 << 63, 1<<63 - 1
	}
	maximum := int64(1)<<(bits-1) - 1
	return -maximum - 1, maximum
}

func unsignedMaximum(bits int) uint64 {
	if bits == 64 {
		return ^uint64(0)
	}
	return uint64(1)<<bits - 1
}
