package analyzer

import (
	"go/types"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rhevorn/shape/internal/jsonfields"
	"github.com/rhevorn/shape/internal/spec"
	"github.com/rhevorn/shape/internal/taglang"
)

func checkStructType(t types.Type, active map[types.Type]bool) error {
	t = types.Unalias(t)
	if classify(t) == kTime {
		return errText("target must be an ordinary value struct")
	}
	if named, ok := t.(*types.Named); ok {
		t = named.Underlying()
	}
	value, ok := t.Underlying().(*types.Struct)
	if !ok {
		return errText("target must be a value struct")
	}
	if active[t] {
		return errText("recursive type is unsupported")
	}
	active[t] = true
	defer delete(active, t)
	names := make(map[string]bool, value.NumFields())
	for index := 0; index < value.NumFields(); index++ {
		field := value.Field(index)
		tag := reflect.StructTag(value.Tag(index))
		shapeTag := tag.Get("shape")
		jsonName, _ := jsonfields.Name(field.Name(), tag.Get("json"))
		if !field.Exported() || tag.Get("json") == "-" {
			if shapeTag != "" {
				return errText(field.Name() + " has a tag but is not processed")
			}
			continue
		}
		if field.Embedded() {
			return errText("anonymous fields are unsupported")
		}
		if jsonName == "" {
			jsonName = field.Name()
		}
		if names[jsonName] {
			return errText("duplicate field name " + jsonName)
		}
		names[jsonName] = true
		items, err := taglang.Parse(shapeTag)
		if err != nil {
			return errText("field " + field.Name() + ": " + err.Error())
		}
		if err := checkTag(field.Type(), items); err != nil {
			return errText("field " + field.Name() + ": " + err.Error())
		}
		if err := checkSupportedGraph(field.Type(), active); err != nil {
			return errText("field " + field.Name() + ": " + err.Error())
		}
	}
	return nil
}

func checkSupportedGraph(t types.Type, active map[types.Type]bool) error {
	t = types.Unalias(t)
	if classify(t) == kStruct {
		return checkStructType(t, active)
	}
	if active[t] {
		return errText("recursive type is unsupported")
	}
	active[t] = true
	defer delete(active, t)
	switch value := t.(type) {
	case *types.Pointer:
		element := types.Unalias(value.Elem())
		if classify(element) == kPointer {
			return errText("multi-level pointers are unsupported in tags")
		}
		if classify(element) == kSlice || classify(element) == kMap {
			return errText("pointers to slices and maps are unsupported in tags")
		}
		return checkSupportedGraph(element, active)
	case *types.Named:
		switch classify(value) {
		case kTime, kDuration, kString, kBool, kNumber:
			return nil
		case kStruct:
			return checkStructType(value, active)
		default:
			return checkSupportedGraph(value.Underlying(), active)
		}
	}
	switch value := t.Underlying().(type) {
	case *types.Struct:
		return checkStructType(value, active)
	case *types.Slice:
		return checkSupportedGraph(value.Elem(), active)
	case *types.Map:
		if !isMapKey(value.Key()) {
			return errText("map key must be string or integer")
		}
		return checkSupportedGraph(value.Elem(), active)
	case *types.Basic:
		if classify(value) == kUnsupported {
			return errText("unsupported field type")
		}
		return nil
	default:
		return errText("unsupported field type")
	}
}

func checkTag(t types.Type, items []taglang.Item) error {
	fieldKind := classify(t)
	if fieldKind == kUnsupported {
		return errText("unsupported field type")
	}
	target := fieldKind
	if fieldKind == kPointer {
		target = classify(element(t))
		if target == kUnsupported || target == kPointer || target == kSlice || target == kMap {
			return errText("unsupported pointer field type")
		}
	}
	fallbackSeen, labelSeen := false, false
	for _, item := range items {
		if item.Name == "label" {
			if !item.HasValue || labelSeen {
				return errText("label requires one value and may appear once")
			}
			labelSeen = true
		}
		if item.Name == "ifzero" || item.Name == "ifnull" {
			if !item.HasValue || fallbackSeen {
				return errText("invalid or duplicate fallback")
			}
			fallbackSeen = true
		}
		if err := spec.CheckTag(item.Name, domain(fieldKind), domain(target), item.HasValue); err != nil {
			return err
		}

		if item.HasValue {
			valueType := t
			if fieldKind == kPointer {
				valueType = element(t)
			}
			if err := checkTagValue(valueType, target, item); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkTagValue(typ types.Type, target kind, item taglang.Item) error {
	switch item.Name {
	case "label", "trim", "ltrim", "rtrim", "startswith", "endswith", "contains":
		return nil
	case "pattern":
		if item.Value == "" {
			return errText("pattern must not be empty")
		}
		if _, err := regexp.Compile(item.Value); err != nil {
			return errText("invalid pattern: " + err.Error())
		}
		return nil
	case "minlength", "maxlength", "len":
		if _, err := strconv.ParseUint(item.Value, 10, 63); err != nil {
			return errText("invalid non-negative length")
		}
		return nil
	}
	if item.Name == "ifzero" || item.Name == "ifnull" {
		switch target {
		case kString:
			return nil
		case kBool:
			if item.Value != "true" && item.Value != "false" {
				return errText("boolean fallback must be true or false")
			}
			return nil
		case kDuration:
			if item.Value == "0" {
				return nil
			}
			if _, err := time.ParseDuration(item.Value); err != nil {
				return errText("invalid duration " + strconv.Quote(item.Value))
			}
			return nil
		case kTime:
			if _, err := time.Parse(time.RFC3339Nano, item.Value); err != nil {
				return errText("invalid RFC3339 time")
			}
			return nil
		case kNumber:
			return checkNumber(typ, item.Value)
		}
	}
	if (target == kSlice || target == kMap) && (item.Name == "min" || item.Name == "max") {
		if _, err := strconv.ParseUint(item.Value, 10, 63); err != nil {
			return errText("invalid non-negative collection length")
		}
		return nil
	}
	parts := []string{item.Value}
	if item.Name == "between" || item.Name == "oneof" {
		parts = strings.Split(item.Value, "|")
		if item.Name == "between" && len(parts) != 2 || item.Name == "oneof" && len(parts) == 0 {
			return errText("invalid list arity")
		}
	}
	if item.Name == "between" && len(parts) == 2 && greaterValue(target, parts[0], parts[1]) {
		return errText("between minimum exceeds maximum")
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			if target == kString && item.Name == "oneof" {
				continue
			}
			return errText("empty list value")
		}
		if target == kDuration {
			if part != "0" {
				if _, err := time.ParseDuration(part); err != nil {
					return errText("invalid duration " + strconv.Quote(part))
				}
			}
			continue
		}
		if target == kNumber {
			if err := checkNumber(typ, part); err != nil {
				return err
			}
		}
	}
	return nil
}

type errText string

func (e errText) Error() string { return string(e) }

func domain(k kind) spec.Kind {
	switch k {
	case kString:
		return spec.String
	case kBool:
		return spec.Bool
	case kNumber:
		return spec.Number
	case kDuration:
		return spec.Duration
	case kTime:
		return spec.Time
	case kPointer:
		return spec.Pointer
	case kSlice:
		return spec.Slice
	case kMap:
		return spec.Map
	case kStruct:
		return spec.Struct
	}
	return 0
}
