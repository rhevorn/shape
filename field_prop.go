package shape

import "regexp"

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
