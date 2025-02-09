package shape

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"
)

var (
	structDurationType = reflect.TypeOf(time.Duration(0))
	structTimeType     = reflect.TypeOf(time.Time{})
	structSchemaCache  sync.Map // reflect.Type -> ObjectSchema[T]
)

// Struct builds an ObjectSchema[T] from exported struct fields using `json`
// names and `shape` rule tags. Successful results are cached by type so later
// Struct / MustStruct / Bind calls reuse the same schema. Reflection runs only
// on a cache miss; parse still uses the normal Object path.
//
// Tag names match fluent methods in lowercase (Trim -> trim, Min -> min, ...).
// Field options: label='...', optional. json:",omitempty" also marks optional.
// On time.Duration / time.Time, coerce selects CoerceDuration / CoerceTime.
//
// Supported field kinds: string, bool, int, int64, float64, time.Duration,
// time.Time, nested struct, and slices of those kinds. Pointer fields are not
// supported.
func Struct[T any]() (ObjectSchema[T], error) {
	typ := reflect.TypeFor[T]()
	if typ == nil {
		return ObjectSchema[T]{}, fmt.Errorf("shape: Struct type is invalid")
	}
	if typ.Kind() == reflect.Pointer {
		return ObjectSchema[T]{}, fmt.Errorf("shape: Struct expects a struct type, got pointer %s", typ)
	}
	if typ.Kind() != reflect.Struct {
		return ObjectSchema[T]{}, fmt.Errorf("shape: Struct expects a struct type, got %s", typ.Kind())
	}
	if cached, ok := structSchemaCache.Load(typ); ok {
		return objectSchemaFromCache[T](cached), nil
	}
	fields, err := structFields[T](typ, nil)
	if err != nil {
		return ObjectSchema[T]{}, err
	}
	schema := Object(fields...)
	if actual, loaded := structSchemaCache.LoadOrStore(typ, schema); loaded {
		return objectSchemaFromCache[T](actual), nil
	}
	return schema, nil
}

// objectSchemaFromCache copies a cached ObjectSchema without a type-parameter
// type assertion (which some editors fail to highlight).
func objectSchemaFromCache[T any](cached any) ObjectSchema[T] {
	var out ObjectSchema[T]
	reflect.ValueOf(&out).Elem().Set(reflect.ValueOf(cached))
	return out
}

// MustStruct is Struct but panics on configuration errors.
func MustStruct[T any]() ObjectSchema[T] {
	schema, err := Struct[T]()
	if err != nil {
		panic(err)
	}
	return schema
}

func structFields[T any](typ reflect.Type, indexPrefix []int) ([]ObjectField[T], error) {
	fields := make([]ObjectField[T], 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		index := append(append([]int{}, indexPrefix...), i)
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
			embedded, err := structFields[T](sf.Type, index)
			if err != nil {
				return nil, err
			}
			fields = append(fields, embedded...)
			continue
		}
		name, skip, jsonOmitEmpty := jsonFieldName(sf)
		if skip {
			continue
		}
		options, err := parseShapeTag(sf.Tag.Get("shape"))
		if err != nil {
			return nil, fmt.Errorf("shape: field %s: %w", sf.Name, err)
		}
		field, err := buildStructField[T](name, sf.Type, index, options, jsonOmitEmpty)
		if err != nil {
			return nil, fmt.Errorf("shape: field %s: %w", sf.Name, err)
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func jsonFieldName(sf reflect.StructField) (name string, skip bool, omitEmpty bool) {
	tag := sf.Tag.Get("json")
	if tag == "-" {
		return "", true, false
	}
	if tag == "" {
		return sf.Name, false, false
	}
	parts := splitComma(tag)
	name = parts[0]
	if name == "" {
		name = sf.Name
	}
	for _, part := range parts[1:] {
		if part == "omitempty" {
			omitEmpty = true
		}
	}
	return name, false, omitEmpty
}

func splitComma(tag string) []string {
	parts := make([]string, 0, 4)
	start := 0
	for i := 0; i <= len(tag); i++ {
		if i == len(tag) || tag[i] == ',' {
			parts = append(parts, tag[start:i])
			start = i + 1
		}
	}
	return parts
}

func buildStructField[T any](
	name string,
	fieldType reflect.Type,
	index []int,
	options []tagOption,
	jsonOmitEmpty bool,
) (ObjectField[T], error) {
	label, optional, filtered, err := takeFieldOptions(options, jsonOmitEmpty)
	if err != nil {
		return nil, err
	}

	switch {
	case fieldType == structDurationType:
		schema, err := applyDurationTags(Duration(), filtered)
		if err != nil {
			return nil, err
		}
		return finishStructField[T, time.Duration](name, label, optional, schema, index), nil
	case fieldType == structTimeType:
		schema, err := applyTimeTags(Time(), filtered)
		if err != nil {
			return nil, err
		}
		return finishStructField[T, time.Time](name, label, optional, schema, index), nil
	}

	switch fieldType.Kind() {
	case reflect.String:
		schema, err := applyStringTags(String(), filtered)
		if err != nil {
			return nil, err
		}
		return finishStructField[T, string](name, label, optional, schema, index), nil
	case reflect.Bool:
		if len(filtered) != 0 {
			return nil, fmt.Errorf("unsupported bool tag %q", filtered[0].name)
		}
		return finishStructField[T, bool](name, label, optional, Bool(), index), nil
	case reflect.Int:
		schema, err := applyIntTags(Int(), filtered)
		if err != nil {
			return nil, err
		}
		return finishStructField[T, int](name, label, optional, schema, index), nil
	case reflect.Int64:
		schema, err := applyInt64Tags(Int64(), filtered)
		if err != nil {
			return nil, err
		}
		return finishStructField[T, int64](name, label, optional, schema, index), nil
	case reflect.Float64:
		schema, err := applyFloat64Tags(Float64(), filtered)
		if err != nil {
			return nil, err
		}
		return finishStructField[T, float64](name, label, optional, schema, index), nil
	case reflect.Struct:
		if len(filtered) != 0 {
			return nil, fmt.Errorf("nested struct does not accept value tags (got %q)", filtered[0].name)
		}
		nested, err := reflectStructSchema(fieldType)
		if err != nil {
			return nil, err
		}
		return finishStructField[T, any](name, label, optional, nested, index), nil
	case reflect.Slice:
		schema, err := applySliceTags(fieldType.Elem(), filtered)
		if err != nil {
			return nil, err
		}
		return finishStructField[T, any](name, label, optional, schema, index), nil
	case reflect.Pointer:
		return nil, fmt.Errorf("pointer fields are not supported; use optional on a value field")
	default:
		return nil, fmt.Errorf("unsupported field type %s", fieldType)
	}
}

func takeFieldOptions(options []tagOption, jsonOmitEmpty bool) (label string, optional bool, rest []tagOption, err error) {
	optional = jsonOmitEmpty
	rest = make([]tagOption, 0, len(options))
	for _, opt := range options {
		switch opt.name {
		case "label":
			label, err = tagRequiredString(opt)
			if err != nil {
				return "", false, nil, err
			}
		case "optional":
			if opt.has {
				return "", false, nil, fmt.Errorf("tag optional takes no value")
			}
			optional = true
		default:
			rest = append(rest, opt)
		}
	}
	return label, optional, rest, nil
}

func finishStructField[T, V any](name, label string, optional bool, schema Schema[V], index []int) ObjectField[T] {
	index = append([]int(nil), index...)
	field := Field(name, schema, func(target *T, value V) {
		field := reflect.ValueOf(target).Elem().FieldByIndex(index)
		rv := reflect.ValueOf(value)
		if !rv.IsValid() {
			field.Set(reflect.Zero(field.Type()))
			return
		}
		if rv.Type().AssignableTo(field.Type()) {
			field.Set(rv)
			return
		}
		field.Set(rv.Convert(field.Type()))
	})
	if label != "" {
		field = field.Label(label)
	}
	if optional {
		field = field.Optional()
	}
	return field
}

// reflectStructSchema builds a Schema[any] that parses into a value of typ.
func reflectStructSchema(typ reflect.Type) (Schema[any], error) {
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", typ.Kind())
	}
	fields, err := reflectStructFields(typ, nil)
	if err != nil {
		return nil, err
	}
	return reflectObjectSchema{typ: typ, fields: fields}, nil
}

type reflectObjectField struct {
	name     string
	label    string
	optional bool
	schema   Schema[any]
	index    []int
}

type reflectObjectSchema struct {
	typ    reflect.Type
	fields []reflectObjectField
}

func (s reflectObjectSchema) Parse(value any) (any, error) {
	return s.ParseContext(context.Background(), value)
}

func (s reflectObjectSchema) ParseContext(ctx context.Context, value any) (any, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	input, ok := value.(map[string]any)
	if !ok {
		return nil, validationError(ctx, invalidType("map[string]any", value))
	}
	out := reflect.New(s.typ).Elem()
	var issues []Issue
	for _, field := range s.fields {
		fieldValue, present := input[field.name]
		if !present {
			if field.optional {
				continue
			}
			issue := keyedIssue(CodeRequired, "required", "present", "missing")
			issue.Path = Path{FieldPath(field.name)}
			issue.Label = field.label
			var capped bool
			issues, capped = appendIssuesBounded(issues, issue)
			if capped {
				break
			}
			continue
		}
		parsed, err := field.schema.ParseContext(ctx, fieldValue)
		if err != nil {
			if contextErr := contextError(err, ctx); contextErr != nil {
				return nil, contextErr
			}
			var capped bool
			issues, capped = appendIssuesBounded(issues, applyIssueLabel(prefixIssues(issuesFromError(err), FieldPath(field.name)), field.label)...)
			if capped {
				break
			}
			continue
		}
		fv := out.FieldByIndex(field.index)
		rv := reflect.ValueOf(parsed)
		if !rv.IsValid() {
			fv.Set(reflect.Zero(fv.Type()))
			continue
		}
		if rv.Type().AssignableTo(fv.Type()) {
			fv.Set(rv)
		} else {
			fv.Set(rv.Convert(fv.Type()))
		}
	}
	if len(issues) != 0 {
		return nil, validationIssues(ctx, issues)
	}
	return out.Interface(), nil
}

func (s reflectObjectSchema) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	properties := make(map[string]any, len(s.fields))
	required := make([]string, 0, len(s.fields))
	for _, field := range s.fields {
		document, err := buildJSONSchemaWithContext(field.schema, ctx)
		if err != nil {
			return nil, err
		}
		properties[field.name] = document
		if !field.optional {
			required = append(required, field.name)
		}
	}
	document := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": true,
	}
	if len(required) != 0 {
		document["required"] = required
	}
	return document, nil
}

func reflectStructFields(typ reflect.Type, indexPrefix []int) ([]reflectObjectField, error) {
	fields := make([]reflectObjectField, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		index := append(append([]int{}, indexPrefix...), i)
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
			embedded, err := reflectStructFields(sf.Type, index)
			if err != nil {
				return nil, err
			}
			fields = append(fields, embedded...)
			continue
		}
		name, skip, jsonOmitEmpty := jsonFieldName(sf)
		if skip {
			continue
		}
		options, err := parseShapeTag(sf.Tag.Get("shape"))
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", sf.Name, err)
		}
		label, optional, filtered, err := takeFieldOptions(options, jsonOmitEmpty)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", sf.Name, err)
		}
		schema, err := schemaForType(sf.Type, filtered)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", sf.Name, err)
		}
		fields = append(fields, reflectObjectField{
			name:     name,
			label:    label,
			optional: optional,
			schema:   schema,
			index:    index,
		})
	}
	return fields, nil
}

func schemaForType(typ reflect.Type, options []tagOption) (Schema[any], error) {
	switch {
	case typ == structDurationType:
		schema, err := applyDurationTags(Duration(), options)
		if err != nil {
			return nil, err
		}
		return asAny(schema), nil
	case typ == structTimeType:
		schema, err := applyTimeTags(Time(), options)
		if err != nil {
			return nil, err
		}
		return asAny(schema), nil
	}

	switch typ.Kind() {
	case reflect.String:
		schema, err := applyStringTags(String(), options)
		if err != nil {
			return nil, err
		}
		return asAny(schema), nil
	case reflect.Bool:
		if len(options) != 0 {
			return nil, fmt.Errorf("unsupported bool tag %q", options[0].name)
		}
		return asAny(Bool()), nil
	case reflect.Int:
		schema, err := applyIntTags(Int(), options)
		if err != nil {
			return nil, err
		}
		return asAny(schema), nil
	case reflect.Int64:
		schema, err := applyInt64Tags(Int64(), options)
		if err != nil {
			return nil, err
		}
		return asAny(schema), nil
	case reflect.Float64:
		schema, err := applyFloat64Tags(Float64(), options)
		if err != nil {
			return nil, err
		}
		return asAny(schema), nil
	case reflect.Struct:
		if len(options) != 0 {
			return nil, fmt.Errorf("nested struct does not accept value tags (got %q)", options[0].name)
		}
		return reflectStructSchema(typ)
	case reflect.Slice:
		return applySliceTags(typ.Elem(), options)
	case reflect.Pointer:
		return nil, fmt.Errorf("pointer fields are not supported; use optional on a value field")
	default:
		return nil, fmt.Errorf("unsupported field type %s", typ)
	}
}

func asAny[V any](schema Schema[V]) Schema[any] {
	return anyBox[V]{schema: schema}
}

// anyBox lifts Schema[V] to Schema[any] while preserving JSON Schema export.
type anyBox[V any] struct {
	schema Schema[V]
}

func (s anyBox[V]) Parse(value any) (any, error) {
	return s.ParseContext(context.Background(), value)
}

func (s anyBox[V]) ParseContext(ctx context.Context, value any) (any, error) {
	parsed, err := s.schema.ParseContext(ctx, value)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func (s anyBox[V]) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	return buildJSONSchemaWithContext(s.schema, ctx)
}

func applySliceTags(elem reflect.Type, options []tagOption) (Schema[any], error) {
	elemSchema, err := schemaForType(elem, nil)
	if err != nil {
		return nil, err
	}
	slice := Slice(elemSchema)
	for _, opt := range options {
		switch opt.name {
		case "min":
			n, err := tagInt(opt)
			if err != nil {
				return nil, err
			}
			slice = slice.Min(n)
		case "max":
			n, err := tagInt(opt)
			if err != nil {
				return nil, err
			}
			slice = slice.Max(n)
		case "nonempty":
			if opt.has {
				return nil, fmt.Errorf("tag nonempty takes no value")
			}
			slice = slice.NonEmpty()
		case "unique":
			if opt.has {
				return nil, fmt.Errorf("tag unique takes no value")
			}
			slice = slice.Unique()
		default:
			return nil, fmt.Errorf("unsupported slice tag %q", opt.name)
		}
	}
	return concreteSliceSchema{elemType: elem, slice: slice}, nil
}

// concreteSliceSchema parses []any elements then rebuilds a typed Go slice.
type concreteSliceSchema struct {
	elemType reflect.Type
	slice    SliceSchema[any]
}

func (s concreteSliceSchema) Parse(value any) (any, error) {
	return s.ParseContext(context.Background(), value)
}

func (s concreteSliceSchema) ParseContext(ctx context.Context, value any) (any, error) {
	items, err := s.slice.ParseContext(ctx, value)
	if err != nil {
		return nil, err
	}
	out := reflect.MakeSlice(reflect.SliceOf(s.elemType), len(items), len(items))
	for i, item := range items {
		rv := reflect.ValueOf(item)
		if !rv.IsValid() {
			out.Index(i).Set(reflect.Zero(s.elemType))
			continue
		}
		if rv.Type().AssignableTo(s.elemType) {
			out.Index(i).Set(rv)
		} else {
			out.Index(i).Set(rv.Convert(s.elemType))
		}
	}
	return out.Interface(), nil
}

func (s concreteSliceSchema) buildJSONSchema(ctx *jsonSchemaBuildContext) (map[string]any, error) {
	return s.slice.buildJSONSchema(ctx)
}

// Bind decodes JSON into dest using a cached MustStruct[T]().Strict() schema.
// Prefer an explicit schema with Parse* when you need Strip, Refine, or a
// hand-written Object.
func Bind[T any, B jsonText](dest *T, data B) error {
	return BindContext(context.Background(), dest, data)
}

// BindContext is Bind with context propagation.
func BindContext[T any, B jsonText](ctx context.Context, dest *T, data B) error {
	if dest == nil {
		return errors.New("shape: bind destination must not be nil")
	}
	value, err := ParseContext(ctx, MustStruct[T]().Strict(), data)
	if err != nil {
		return err
	}
	*dest = value
	return nil
}

// BindReaderLimit binds JSON from a size-limited reader into dest.
func BindReaderLimit[T any](dest *T, reader io.Reader, maxBytes int64) error {
	return BindReaderLimitContext(context.Background(), dest, reader, maxBytes)
}

// BindReaderLimitContext is BindReaderLimit with context propagation.
//
//	var req CreateUserRequest
//	err := BindReaderLimitContext(ctx, &req, body, 1<<20)
func BindReaderLimitContext[T any](ctx context.Context, dest *T, reader io.Reader, maxBytes int64) error {
	if dest == nil {
		return errors.New("shape: bind destination must not be nil")
	}
	value, err := ParseReaderLimitContext(ctx, MustStruct[T]().Strict(), reader, maxBytes)
	if err != nil {
		return err
	}
	*dest = value
	return nil
}
