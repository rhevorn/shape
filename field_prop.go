package shape

import (
	"regexp"
	"time"
)

func optionalFieldLabel(label []string) string {
	switch len(label) {
	case 0:
		return ""
	case 1:
		return label[0]
	default:
		panic("shape: field label accepts at most one value")
	}
}

func finishField[T, V any](key, label string, schema Schema[V], setter func(*T, V)) FieldDef[T, V] {
	field := Field(key, schema, setter)
	if label != "" {
		field = field.Label(label)
	}
	return field
}

// Fields returns a field factory for object type T. Use it to keep field
// helpers short while preserving type inference:
//
//	f := shape.Fields[User]()
//	shape.Object(
//		f.Str("name", "姓名").Trim().Min(2).Set(func(user *User, value string) {
//			user.Name = value
//		}),
//	)
func Fields[T any]() FieldFactory[T] { return FieldFactory[T]{} }

// FieldFactory builds typed object fields for T.
type FieldFactory[T any] struct{}

// Str starts an object string field. The optional label is a display name.
func (FieldFactory[T]) Str(key string, label ...string) StringField[T] {
	return StringField[T]{key: key, label: optionalFieldLabel(label), schema: String()}
}

// Email starts an object email string field.
func (f FieldFactory[T]) Email(key string, label ...string) StringField[T] {
	return f.Str(key, label...).Email()
}

// Int starts an object int field. The optional label is a display name.
func (FieldFactory[T]) Int(key string, label ...string) IntObjectField[T] {
	return IntObjectField[T]{key: key, label: optionalFieldLabel(label), schema: Int()}
}

// Bool starts an object bool field. The optional label is a display name.
func (FieldFactory[T]) Bool(key string, label ...string) BoolObjectField[T] {
	return BoolObjectField[T]{key: key, label: optionalFieldLabel(label), schema: Bool()}
}

// Int64 starts an object int64 field. The optional label is a display name.
func (FieldFactory[T]) Int64(key string, label ...string) Int64ObjectField[T] {
	return Int64ObjectField[T]{key: key, label: optionalFieldLabel(label), schema: Int64()}
}

// Float64 starts an object float64 field. The optional label is a display name.
func (FieldFactory[T]) Float64(key string, label ...string) Float64ObjectField[T] {
	return Float64ObjectField[T]{key: key, label: optionalFieldLabel(label), schema: Float64()}
}

// Time starts a strict object time.Time field. The optional label is a display name.
func (FieldFactory[T]) Time(key string, label ...string) TimeObjectField[T] {
	return TimeObjectField[T]{key: key, label: optionalFieldLabel(label), schema: Time()}
}

// Duration starts a strict object time.Duration field. The optional label is a display name.
func (FieldFactory[T]) Duration(key string, label ...string) DurationObjectField[T] {
	return DurationObjectField[T]{key: key, label: optionalFieldLabel(label), schema: Duration()}
}

// StringField is a fluent object-field builder for string values.
type StringField[T any] struct {
	key    string
	label  string
	schema StringSchema
}

func (f StringField[T]) Trim() StringField[T] {
	f.schema = f.schema.Trim()
	return f
}

func (f StringField[T]) ToLower() StringField[T] {
	f.schema = f.schema.ToLower()
	return f
}

func (f StringField[T]) ToUpper() StringField[T] {
	f.schema = f.schema.ToUpper()
	return f
}

func (f StringField[T]) NonEmpty() StringField[T] {
	f.schema = f.schema.NonEmpty()
	return f
}

func (f StringField[T]) Min(n int) StringField[T] {
	f.schema = f.schema.Min(n)
	return f
}

func (f StringField[T]) Max(n int) StringField[T] {
	f.schema = f.schema.Max(n)
	return f
}

func (f StringField[T]) Len(n int) StringField[T] {
	f.schema = f.schema.Len(n)
	return f
}

func (f StringField[T]) Pattern(pattern *regexp.Regexp) StringField[T] {
	f.schema = f.schema.Pattern(pattern)
	return f
}

func (f StringField[T]) StartsWith(prefix string) StringField[T] {
	f.schema = f.schema.StartsWith(prefix)
	return f
}

func (f StringField[T]) EndsWith(suffix string) StringField[T] {
	f.schema = f.schema.EndsWith(suffix)
	return f
}

func (f StringField[T]) Contains(part string) StringField[T] {
	f.schema = f.schema.Contains(part)
	return f
}

func (f StringField[T]) Email() StringField[T] {
	f.schema = f.schema.Email()
	return f
}

func (f StringField[T]) URL() StringField[T] {
	f.schema = f.schema.URL()
	return f
}

func (f StringField[T]) UUID() StringField[T] {
	f.schema = f.schema.UUID()
	return f
}

func (f StringField[T]) IP() StringField[T] {
	f.schema = f.schema.IP()
	return f
}

func (f StringField[T]) Refine(fn func(string) error) StringField[T] {
	f.schema = f.schema.Refine(fn)
	return f
}

// Set finishes the field and binds the parsed value onto T.
func (f StringField[T]) Set(setter func(*T, string)) FieldDef[T, string] {
	return finishField(f.key, f.label, f.schema, setter)
}

// IntObjectField is a fluent object-field builder for int values.
type IntObjectField[T any] struct {
	key    string
	label  string
	schema IntSchema
}

func (f IntObjectField[T]) Min(bound int) IntObjectField[T] {
	f.schema = f.schema.Min(bound)
	return f
}

func (f IntObjectField[T]) Max(bound int) IntObjectField[T] {
	f.schema = f.schema.Max(bound)
	return f
}

func (f IntObjectField[T]) Gt(bound int) IntObjectField[T] {
	f.schema = f.schema.Gt(bound)
	return f
}

func (f IntObjectField[T]) Gte(bound int) IntObjectField[T] {
	f.schema = f.schema.Gte(bound)
	return f
}

func (f IntObjectField[T]) Lt(bound int) IntObjectField[T] {
	f.schema = f.schema.Lt(bound)
	return f
}

func (f IntObjectField[T]) Lte(bound int) IntObjectField[T] {
	f.schema = f.schema.Lte(bound)
	return f
}

func (f IntObjectField[T]) Positive() IntObjectField[T] {
	f.schema = f.schema.Positive()
	return f
}

func (f IntObjectField[T]) Negative() IntObjectField[T] {
	f.schema = f.schema.Negative()
	return f
}

func (f IntObjectField[T]) NonNegative() IntObjectField[T] {
	f.schema = f.schema.NonNegative()
	return f
}

func (f IntObjectField[T]) OneOf(values ...int) IntObjectField[T] {
	f.schema = f.schema.OneOf(values...)
	return f
}

func (f IntObjectField[T]) Refine(fn func(int) error) IntObjectField[T] {
	f.schema = f.schema.Refine(fn)
	return f
}

// Set finishes the field and binds the parsed value onto T.
func (f IntObjectField[T]) Set(setter func(*T, int)) FieldDef[T, int] {
	return finishField(f.key, f.label, f.schema, setter)
}

// BoolObjectField is a fluent object-field builder for bool values.
type BoolObjectField[T any] struct {
	key    string
	label  string
	schema BoolSchema
}

func (f BoolObjectField[T]) Refine(fn func(bool) error) BoolObjectField[T] {
	f.schema = f.schema.Refine(fn)
	return f
}

// Set finishes the field and binds the parsed value onto T.
func (f BoolObjectField[T]) Set(setter func(*T, bool)) FieldDef[T, bool] {
	return finishField(f.key, f.label, f.schema, setter)
}

// Int64ObjectField is a fluent object-field builder for int64 values.
type Int64ObjectField[T any] struct {
	key    string
	label  string
	schema Int64Schema
}

func (f Int64ObjectField[T]) Min(bound int64) Int64ObjectField[T] {
	f.schema = f.schema.Min(bound)
	return f
}

func (f Int64ObjectField[T]) Max(bound int64) Int64ObjectField[T] {
	f.schema = f.schema.Max(bound)
	return f
}

func (f Int64ObjectField[T]) Gt(bound int64) Int64ObjectField[T] {
	f.schema = f.schema.Gt(bound)
	return f
}

func (f Int64ObjectField[T]) Gte(bound int64) Int64ObjectField[T] {
	f.schema = f.schema.Gte(bound)
	return f
}

func (f Int64ObjectField[T]) Lt(bound int64) Int64ObjectField[T] {
	f.schema = f.schema.Lt(bound)
	return f
}

func (f Int64ObjectField[T]) Lte(bound int64) Int64ObjectField[T] {
	f.schema = f.schema.Lte(bound)
	return f
}

func (f Int64ObjectField[T]) Positive() Int64ObjectField[T] {
	f.schema = f.schema.Positive()
	return f
}

func (f Int64ObjectField[T]) Negative() Int64ObjectField[T] {
	f.schema = f.schema.Negative()
	return f
}

func (f Int64ObjectField[T]) NonNegative() Int64ObjectField[T] {
	f.schema = f.schema.NonNegative()
	return f
}

func (f Int64ObjectField[T]) OneOf(values ...int64) Int64ObjectField[T] {
	f.schema = f.schema.OneOf(values...)
	return f
}

func (f Int64ObjectField[T]) Refine(fn func(int64) error) Int64ObjectField[T] {
	f.schema = f.schema.Refine(fn)
	return f
}

// Set finishes the field and binds the parsed value onto T.
func (f Int64ObjectField[T]) Set(setter func(*T, int64)) FieldDef[T, int64] {
	return finishField(f.key, f.label, f.schema, setter)
}

// Float64ObjectField is a fluent object-field builder for float64 values.
type Float64ObjectField[T any] struct {
	key    string
	label  string
	schema Float64Schema
}

func (f Float64ObjectField[T]) Min(bound float64) Float64ObjectField[T] {
	f.schema = f.schema.Min(bound)
	return f
}

func (f Float64ObjectField[T]) Max(bound float64) Float64ObjectField[T] {
	f.schema = f.schema.Max(bound)
	return f
}

func (f Float64ObjectField[T]) Gt(bound float64) Float64ObjectField[T] {
	f.schema = f.schema.Gt(bound)
	return f
}

func (f Float64ObjectField[T]) Gte(bound float64) Float64ObjectField[T] {
	f.schema = f.schema.Gte(bound)
	return f
}

func (f Float64ObjectField[T]) Lt(bound float64) Float64ObjectField[T] {
	f.schema = f.schema.Lt(bound)
	return f
}

func (f Float64ObjectField[T]) Lte(bound float64) Float64ObjectField[T] {
	f.schema = f.schema.Lte(bound)
	return f
}

func (f Float64ObjectField[T]) Positive() Float64ObjectField[T] {
	f.schema = f.schema.Positive()
	return f
}

func (f Float64ObjectField[T]) Negative() Float64ObjectField[T] {
	f.schema = f.schema.Negative()
	return f
}

func (f Float64ObjectField[T]) NonNegative() Float64ObjectField[T] {
	f.schema = f.schema.NonNegative()
	return f
}

func (f Float64ObjectField[T]) OneOf(values ...float64) Float64ObjectField[T] {
	f.schema = f.schema.OneOf(values...)
	return f
}

func (f Float64ObjectField[T]) Refine(fn func(float64) error) Float64ObjectField[T] {
	f.schema = f.schema.Refine(fn)
	return f
}

// Set finishes the field and binds the parsed value onto T.
func (f Float64ObjectField[T]) Set(setter func(*T, float64)) FieldDef[T, float64] {
	return finishField(f.key, f.label, f.schema, setter)
}

// TimeObjectField is a fluent object-field builder for time.Time values.
type TimeObjectField[T any] struct {
	key    string
	label  string
	schema TimeSchema
}

func (f TimeObjectField[T]) Refine(fn func(time.Time) error) TimeObjectField[T] {
	f.schema = f.schema.Refine(fn)
	return f
}

// Set finishes the field and binds the parsed value onto T.
func (f TimeObjectField[T]) Set(setter func(*T, time.Time)) FieldDef[T, time.Time] {
	return finishField(f.key, f.label, f.schema, setter)
}

// DurationObjectField is a fluent object-field builder for time.Duration values.
type DurationObjectField[T any] struct {
	key    string
	label  string
	schema DurationSchema
}

func (f DurationObjectField[T]) Refine(fn func(time.Duration) error) DurationObjectField[T] {
	f.schema = f.schema.Refine(fn)
	return f
}

// Set finishes the field and binds the parsed value onto T.
func (f DurationObjectField[T]) Set(setter func(*T, time.Duration)) FieldDef[T, time.Duration] {
	return finishField(f.key, f.label, f.schema, setter)
}
