package tagged

import (
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"
	"time"

	"github.com/rhevorn/shape/validate"
)

// UnsupportedError reports processing that JSON Schema cannot represent
// without changing Shape's runtime behavior.
type UnsupportedError struct{ Feature string }

// Error identifies the behavior that cannot be represented.
func (e *UnsupportedError) Error() string {
	if e == nil {
		return "shape: schema export does not support this schema"
	}
	return "shape: schema export does not support " + e.Feature
}

// Export builds a JSON Schema document from a compiled tag plan.
func Export(p *Plan) (map[string]any, error) { return exportTagPlanAt(p, false) }

// exportTagPlanAt builds the document for p. pointee is true when p sits behind a
// pointer: there a JSON null decodes to the pointer, not to p, so p must not
// offer a null branch of its own.
func exportTagPlanAt(p *Plan, pointee bool) (map[string]any, error) {
	if p == nil {
		return nil, &UnsupportedError{Feature: "uninitialized schema"}
	}
	if p.fallbackKind != "" {
		return nil, &UnsupportedError{Feature: p.fallbackKind + " fallback"}
	}
	if p.hasTransform {
		return nil, &UnsupportedError{Feature: "transform"}
	}
	t := p.typ
	var document map[string]any
	if t == durationType {
		return nil, &UnsupportedError{Feature: "Go duration string representation"}

	}
	if t == reflect.TypeFor[time.Time]() {
		document = map[string]any{"type": "string", "format": "date-time"}
		return applyExportRules(document, p, pointee)
	}
	if hasCustomJSON(t) {
		return nil, &UnsupportedError{Feature: fmt.Sprintf("custom JSON representation for %v", t)}
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
			return nil, &UnsupportedError{Feature: "custom pointer element"}
		}
		inner, err := exportTagPlanAt(p.element, true)
		if err != nil {
			return nil, err
		}
		document = inner
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return nil, &UnsupportedError{Feature: "base64 byte slice"}
		}
		if p.element == nil {
			return nil, &UnsupportedError{Feature: "custom slice element"}
		}
		inner, err := Export(p.element)
		if err != nil {
			return nil, err
		}
		document = map[string]any{"type": "array", "items": inner}
	case reflect.Map:
		if p.element == nil {
			return nil, &UnsupportedError{Feature: "custom map element"}
		}
		if t.Key().Kind() != reflect.String || hasCustomJSON(t.Key()) {
			return nil, &UnsupportedError{Feature: "non-string map key"}
		}
		inner, err := Export(p.element)
		if err != nil {
			return nil, err
		}
		document = map[string]any{"type": "object", "additionalProperties": inner}
	case reflect.Struct:
		if p.fields == nil {
			return nil, &UnsupportedError{Feature: "explicit Object fields"}
		}
		properties := make(map[string]any, len(p.fields))
		required := make([]string, 0, len(p.fields))
		for _, field := range p.fields {
			if field.quoted {
				return nil, &UnsupportedError{Feature: "json string option on " + field.name}
			}
			child, err := Export(field.plan)
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
		return nil, &UnsupportedError{Feature: fmt.Sprintf("schema %v", t)}
	}
	return applyExportRules(document, p, pointee)
}

var (
	jsonMarshalerType   = reflect.TypeFor[json.Marshaler]()
	jsonUnmarshalerType = reflect.TypeFor[json.Unmarshaler]()
	textMarshalerType   = reflect.TypeFor[encoding.TextMarshaler]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
)

// hasCustomJSON reports whether t or *t implements a JSON or text codec.
// These codecs can change the wire representation independently of the Go fields.
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

func applyExportRules(document map[string]any, p *Plan, pointee bool) (map[string]any, error) {
	for _, descriptor := range p.descriptors {
		var key string
		var value any
		name, args := descriptor.Name, descriptor.Args
		switch name {
		case "min":
			if p.typ.Kind() == reflect.String {
				// String length constraints use minlength, not min.
				return nil, &UnsupportedError{Feature: "rule min on " + p.typ.String()}
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
				mergeExportConstraint(document, "minItems", args[0])
				mergeExportConstraint(document, "maxItems", args[0])
			case reflect.Map:
				mergeExportConstraint(document, "minProperties", args[0])
				mergeExportConstraint(document, "maxProperties", args[0])
			default:
				mergeExportConstraint(document, "minLength", args[0])
				mergeExportConstraint(document, "maxLength", args[0])
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
			key, value = "enum", append([]any(nil), args...)
		case "between":
			mergeExportConstraint(document, "minimum", args[0])
			mergeExportConstraint(document, "maximum", args[1])
			continue
		case "pattern":
			pattern, ok := portablePattern(args[0].(string))
			if !ok {
				return nil, &UnsupportedError{Feature: "non-portable Go regular expression"}
			}
			key, value = "pattern", pattern
		case "ip":
			key, value = "anyOf", []any{map[string]any{"format": "ipv4"}, map[string]any{"format": "ipv6"}}
		case "email", "url", "uuid":
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
			return nil, &UnsupportedError{Feature: "rule " + name}
		}
		mergeExportConstraint(document, key, value)
	}
	if !pointee && planAcceptsZero(p) {
		return map[string]any{
			"anyOf": []any{document, map[string]any{"type": "null"}},
		}, nil
	}
	return document, nil
}

func planAcceptsZero(p *Plan) bool {
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

// Numeric bounds intersect without rounding int64/uint64 through float64.
func mergeExportConstraint(document map[string]any, key string, value any) {
	old, exists := document[key]
	if !exists {
		document[key] = value
		return
	}
	switch key {
	case "minimum", "exclusiveMinimum", "minLength", "minItems", "minProperties":
		if exportNumber(value).Cmp(exportNumber(old)) > 0 {
			document[key] = value
		}
		return
	case "maximum", "exclusiveMaximum", "maxLength", "maxItems", "maxProperties":
		if exportNumber(value).Cmp(exportNumber(old)) < 0 {
			document[key] = value
		}
		return
	}
	if reflect.DeepEqual(old, value) {
		return
	}
	document["allOf"] = append(exportConjunction(document), map[string]any{key: value})
}

func exportConjunction(document map[string]any) []any {
	items, _ := document["allOf"].([]any)
	return items
}

func exportNumber(value any) *big.Rat {
	return numericRat(reflect.ValueOf(value))
}

// Parse Go syntax before emitting the portable subset. In particular, Go's $
// is absolute end-of-text; JavaScript's $ also matches before a final newline.
func portablePattern(pattern string) (string, bool) {
	tree, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return "", false
	}
	return exportRegexp(tree)
}

func exportRegexp(tree *syntax.Regexp) (string, bool) {
	if tree.Flags&syntax.FoldCase != 0 {
		return "", false
	}
	parts := make([]string, len(tree.Sub))
	for i, sub := range tree.Sub {
		part, ok := exportRegexp(sub)
		if !ok {
			return "", false
		}
		parts[i] = part
	}
	switch tree.Op {
	case syntax.OpNoMatch:
		return "(?!)", true
	case syntax.OpEmptyMatch:
		return "(?:)", true
	case syntax.OpLiteral:
		for _, r := range tree.Rune {
			if r > 127 {
				return "", false
			}
		}
		return regexp.QuoteMeta(string(tree.Rune)), true
	case syntax.OpCharClass:
		var out strings.Builder
		out.WriteByte('[')
		for i := 0; i < len(tree.Rune); i += 2 {
			low, high := tree.Rune[i], tree.Rune[i+1]
			if high > 127 {
				return "", false
			}
			fmt.Fprintf(&out, `\x%02x`, low)
			if low != high {
				fmt.Fprintf(&out, `-\x%02x`, high)
			}
		}
		out.WriteByte(']')
		return out.String(), true
	case syntax.OpAnyCharNotNL:
		return `[^\n]`, true
	case syntax.OpAnyChar:
		return `[\s\S]`, true
	case syntax.OpBeginText:
		return "^", true
	case syntax.OpEndText:
		return `(?![\s\S])`, true
	case syntax.OpWordBoundary:
		return `\b`, true
	case syntax.OpNoWordBoundary:
		return `\B`, true
	case syntax.OpCapture:
		return "(?:" + parts[0] + ")", true
	case syntax.OpConcat:
		return strings.Join(parts, ""), true
	case syntax.OpAlternate:
		return "(?:" + strings.Join(parts, "|") + ")", true
	case syntax.OpStar:
		return "(?:" + parts[0] + ")*", true
	case syntax.OpPlus:
		return "(?:" + parts[0] + ")+", true
	case syntax.OpQuest:
		return "(?:" + parts[0] + ")?", true
	case syntax.OpRepeat:
		maxCount := ""
		if tree.Max >= 0 {
			maxCount = strconv.Itoa(tree.Max)
		}
		return "(?:" + parts[0] + "){" + strconv.Itoa(tree.Min) + "," + maxCount + "}", true
	default:
		return "", false
	}
}
