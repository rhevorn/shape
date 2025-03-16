package shape

import (
	"context"
	"io"

	"github.com/rhevorn/shape/transform"
	"github.com/rhevorn/shape/validate"
)

// StringSpec defines transform and validation behavior for a string value.
type StringSpec struct {
	name        string
	transformer transform.StringTransformer
	validator   validate.StringValidator
}

func (f StringSpec) fieldDefinition() erasedField {
	return eraseField(f.name, f.transformer, f.validator)
}
func (f StringSpec) Transform(v string) (string, error) { return f.transformer.Transform(v) }
func (f StringSpec) TransformContext(ctx context.Context, v string) (string, error) {
	return f.transformer.TransformContext(ctx, v)
}
func (f StringSpec) Validate(v string) error { return f.validator.Validate(v) }
func (f StringSpec) ValidateContext(ctx context.Context, v string) error {
	return f.validator.ValidateContext(ctx, v)
}
func (f StringSpec) ValidateFirst(v string) error { return f.validator.ValidateFirst(v) }
func (f StringSpec) ValidateFirstContext(ctx context.Context, v string) error {
	return f.validator.ValidateFirstContext(ctx, v)
}

func (f StringSpec) IfZero(v string) StringSpec { f.transformer = f.transformer.IfZero(v); return f }
func (f StringSpec) Trim(chars ...string) StringSpec {
	f.transformer = f.transformer.Trim(chars...)
	return f
}
func (f StringSpec) LTrim(chars ...string) StringSpec {
	f.transformer = f.transformer.LTrim(chars...)
	return f
}
func (f StringSpec) RTrim(chars ...string) StringSpec {
	f.transformer = f.transformer.RTrim(chars...)
	return f
}
func (f StringSpec) ToLower() StringSpec { f.transformer = f.transformer.ToLower(); return f }
func (f StringSpec) ToUpper() StringSpec { f.transformer = f.transformer.ToUpper(); return f }
func (f StringSpec) Apply(values ...func(string) (string, error)) StringSpec {
	f.transformer = f.transformer.Apply(values...)
	return f
}
func (f StringSpec) ApplyContext(values ...func(context.Context, string) (string, error)) StringSpec {
	f.transformer = f.transformer.ApplyContext(values...)
	return f
}

func (f StringSpec) NotEmpty() StringSpec       { f.validator = f.validator.NotEmpty(); return f }
func (f StringSpec) MinLength(n int) StringSpec { f.validator = f.validator.MinLength(n); return f }
func (f StringSpec) MaxLength(n int) StringSpec { f.validator = f.validator.MaxLength(n); return f }
func (f StringSpec) Len(n int) StringSpec       { f.validator = f.validator.Len(n); return f }
func (f StringSpec) OneOf(values ...string) StringSpec {
	f.validator = f.validator.OneOf(values...)
	return f
}
func (f StringSpec) Pattern(expr string) StringSpec {
	f.validator = f.validator.Pattern(expr)
	return f
}
func (f StringSpec) StartsWith(v string) StringSpec {
	f.validator = f.validator.StartsWith(v)
	return f
}
func (f StringSpec) EndsWith(v string) StringSpec { f.validator = f.validator.EndsWith(v); return f }
func (f StringSpec) Contains(v string) StringSpec { f.validator = f.validator.Contains(v); return f }
func (f StringSpec) Email() StringSpec            { f.validator = f.validator.Email(); return f }
func (f StringSpec) URL() StringSpec              { f.validator = f.validator.URL(); return f }
func (f StringSpec) UUID() StringSpec             { f.validator = f.validator.UUID(); return f }
func (f StringSpec) IP() StringSpec               { f.validator = f.validator.IP(); return f }
func (f StringSpec) Refine(values ...func(string) error) StringSpec {
	f.validator = f.validator.Refine(values...)
	return f
}
func (f StringSpec) RefineContext(values ...func(context.Context, string) error) StringSpec {
	f.validator = f.validator.RefineContext(values...)
	return f
}
func (f StringSpec) Label(label string) StringSpec {
	f.validator = f.validator.Label(label)
	return f
}
func (f StringSpec) Pointer() PointerSpec[string] { return pointerSpec(f.name, f) }
func (f StringSpec) Slice() SliceSpec[string]     { return sliceSpec(f.name, f) }

func (f StringSpec) ParseJSON(source []byte, options ...JSONOptions) (string, error) {
	return parseJSON(f, source, options...)
}
func (f StringSpec) ParseJSONContext(ctx context.Context, source []byte, options ...JSONOptions) (string, error) {
	return parseJSONContext(ctx, f, source, options...)
}
func (f StringSpec) ParseJSONReader(reader io.Reader, options ...JSONOptions) (string, error) {
	return parseJSONReader(f, reader, options...)
}
func (f StringSpec) ParseJSONReaderContext(ctx context.Context, reader io.Reader, options ...JSONOptions) (string, error) {
	return parseJSONReaderContext(ctx, f, reader, options...)
}
func (f StringSpec) BindJSON(target *string, source []byte, options ...JSONOptions) error {
	return bindJSON(f, target, source, options...)
}
func (f StringSpec) BindJSONContext(ctx context.Context, target *string, source []byte, options ...JSONOptions) error {
	return bindJSONContext(ctx, f, target, source, options...)
}
func (f StringSpec) BindJSONReader(target *string, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReader(f, target, reader, options...)
}
func (f StringSpec) BindJSONReaderContext(ctx context.Context, target *string, reader io.Reader, options ...JSONOptions) error {
	return bindJSONReaderContext(ctx, f, target, reader, options...)
}
